package drivers

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/johnstarich/sage/client/web"
	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/prompter"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func init() {
	web.Register((&connectorAlly{}).Description(), func(connector web.CredConnector) (web.Connector, error) {
		p, ok := connector.(web.PasswordConnector)
		if !ok {
			return nil, errors.Errorf("Unsupported connector: %T %+v", connector, connector)
		}
		logger, err := zap.NewProduction()
		return &connectorAlly{
			PasswordConnector: p,
			logger:            logger,
		}, err
	})
}

type connectorAlly struct {
	web.PasswordConnector

	logger *zap.Logger
}

func (c *connectorAlly) Description() string {
	return "Ally"
}

func (c *connectorAlly) FID() string {
	return "6157"
}

func (c *connectorAlly) Org() string {
	return "Ally"
}

// resetWindowOpen reassigns `window.open` to a simple redirect to prevent strange download behavior: https://github.com/chromedp/chromedp/issues/588
func resetWindowOpen(ctx context.Context) error {
	var x string
	return chromedp.EvaluateAsDevTools(`
		(function() {
			window.open = (url, windowName, windowFeatures) => {
				window.location = url
				return null
			} 
			return "replaced"
		})()
	`, &x).Do(ctx)
}

func (c *connectorAlly) Statement(start, end time.Time, accountID string, browser web.Browser, prompt prompter.Prompter) (statementResp *ofxgo.Response, statementErr error) {
	const maxStatementTime = 1 * time.Minute
	const maxStatementFetchTime = 2 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), maxStatementTime)
	defer cancel()
	start = start.Local()
	end = end.Local()

	err := c.signIn(ctx, browser, prompt)
	if err != nil {
		return nil, err
	}

	// passed security prompt, shorten timeout
	ctx, cancel = context.WithTimeout(context.Background(), maxStatementFetchTime)
	defer cancel()

	downloadedFiles := make(chan []byte, 1)
	const downloadTimeout = 20 * time.Second
	browser.Download(func(ctx context.Context, download web.DownloadRequest) error {
		downloadCtx, downloadCancel := context.WithTimeout(ctx, downloadTimeout)
		defer downloadCancel()
		data, err := download.Fetch(downloadCtx)
		if err != nil {
			return err
		}
		downloadedFiles <- data
		return nil
	})

	const dateFormat = "Jan 02, 2006"
	var accountNodes []*cdp.Node
	err = browser.Run(ctx,
		chromedp.WaitReady(`document`),
		chromedp.ActionFunc(resetWindowOpen),
		chromedp.Click(`#accounts-menu-item`),
		chromedp.WaitVisible(`.account-list .account-list-number .account-nickname`),
		chromedp.Nodes(`.account-list .account-list-number`, &accountNodes),
		chromedp.ActionFunc(func(ctx context.Context) error {
			last4AccountID := accountID
			if len(last4AccountID) > 4 {
				last4AccountID = last4AccountID[len(last4AccountID)-4:]
			}

			var accountText []string
			for _, accountNode := range accountNodes {
				if len(accountNode.Children) == 2 {
					checkAccountID := accountNode.Children[1].NodeValue
					if strings.HasSuffix(checkAccountID, last4AccountID) {
						return chromedp.MouseClickNode(accountNode).Do(ctx)
					}
					accountText = append(accountText, checkAccountID)
				}
			}
			return errors.Errorf("Account %q not found in options: %#v", accountID, accountText)
		}),

		chromedp.WaitReady(`document`),
		chromedp.WaitVisible(`.transactions-history a[aria-label="Download"]`),
		chromedp.Click(`.transactions-history a[aria-label="Download"]`),
		chromedp.SendKeys(`#select-file-format`, "Quicken"),
		chromedp.SendKeys(`#select-date-range`, "Custom Dates"),
		chromedp.SendKeys(`#downloadEndDate`, end.Format(dateFormat)),
	)
	if err != nil {
		return nil, err
	}

	{
		var submitErr error
		for ; start.Unix() < end.Unix(); start = start.AddDate(0, 0, 1) {
			// if there's an issue with the start date, try later dates
			submitErr = allySubmitDownloadRequest(ctx, browser, start, end, dateFormat)
			if submitErr == nil || !allyIsAccountOpenDateErr(submitErr) {
				break
			}
		}
		if submitErr != nil {
			if allyIsAccountOpenDateErr(submitErr) {
				return nil, nil
			}
			return nil, submitErr
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case file := <-downloadedFiles:
		if len(file) == 0 {
			// For some reason, Ally returns completely blank files in some downloads
			return nil, nil
		}
		resp, err := ofxgo.ParseResponse(bytes.NewReader(file))
		if err != nil && strings.HasPrefix(err.Error(), "Validation failed:") {
			c.logger.Warn("OFX response failed validation", zap.Error(err))
			err = nil
		}
		return resp, errors.Wrap(err, "Failed to parse response")
	}
}

func (c *connectorAlly) signIn(ctx context.Context, browser web.Browser, prompt prompter.Prompter) error {
	err := browser.Run(ctx,
		network.ClearBrowserCookies(),

		// login
		chromedp.ActionFunc(func(ctx context.Context) error {
			// regular chromedp.Navigate fails if a main script has an uncaught exception
			_, _, _, err := page.Navigate("https://secure.ally.com").Do(ctx)
			return err
		}),
		chromedp.WaitReady(`document`),
		chromedp.Click(`#login-username`),
		chromedp.SendKeys(`#login-username`, c.Username()),
		chromedp.Click(`#login-password`),
		chromedp.SendKeys(`#login-password`, string(c.Password())),
		chromedp.SendKeys(`#login-password`, kb.Enter),
	)
	if err != nil {
		return err
	}

	noWait := chromedp.WaitFunc(func(ctx context.Context, frame *cdp.Frame, ids ...cdp.NodeID) (nodes []*cdp.Node, err error) {
		frame.RLock()
		defer frame.RUnlock()
		for _, id := range ids {
			if node, ok := frame.Nodes[id]; ok {
				nodes = append(nodes, node)
			}
		}
		return
	})

	var securityPromptButton []*cdp.Node
	err = browser.Run(ctx,
		chromedp.WaitReady(`document`),
		chromedp.Sleep(5*time.Second),
		chromedp.Nodes(`button[allytmfn="Send Security Code"]`, &securityPromptButton, noWait),
	)
	if err != nil {
		return err
	}
	if len(securityPromptButton) == 0 {
		return nil
	}
	c.logger.Info("Detected security prompt")
	var securityInput string
	return browser.Run(ctx,
		chromedp.Click(`button[allytmfn="Send Security Code"]`),
		chromedp.WaitReady(`document`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			securityInput, err = prompt.PromptText(ctx, "Enter the security code from Ally:")
			return err
		}),
		chromedp.SetValue(`input[allytmfn="Security Code"]`, securityInput), // TODO this piece still doesn't work
		chromedp.Click(`button[type="submit"]`),
		chromedp.WaitReady(`document`),
		chromedp.Click(`#register-device-yes`),
		chromedp.SendKeys(`#register-device-yes`, kb.Enter),
	)
}

func allyIsAccountOpenDateErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Please enter a start date that falls after your account open date.")
}

func allySubmitDownloadRequest(ctx context.Context, browser web.Browser, start, end time.Time, dateFormat string) error {
	var errorNodes []*cdp.Node
	return browser.Run(ctx,
		chromedp.SetValue(`#downloadStartDate`, ""),
		chromedp.SendKeys(`#downloadStartDate`, start.Format(dateFormat)),
		chromedp.Click(`.transactions-history button[type="submit"]`),
		chromedp.Tasks{
			// group these to reduce recorded screenshot count
			chromedp.Sleep(100 * time.Millisecond),
			chromedp.Nodes(`.error-confirmation-list, .error-confirmation-list > *`, &errorNodes),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if len(errorNodes) == 0 {
					return errors.New("Could not detect history form errors")
				}
				var errs sErrors.Errors
				for _, item := range errorNodes {
					if len(item.Children) > 0 {
						errs.ErrIf(
							item.NodeName == "LI" &&
								item.Children[0].NodeName == "#text" &&
								item.Children[0].NodeValue != "",
							item.Children[0].NodeValue,
						)
					}
				}
				return errs.ErrOrNil()
			}),
		},
	)
}

func walkNodes(node *cdp.Node, visit func(node *cdp.Node) (keepGoing bool)) (keepGoing bool) {
	if node == nil {
		return true
	}

	if !visit(node) {
		return false
	}
	for _, child := range node.Children {
		if !walkNodes(child, visit) {
			return false
		}
	}
	return true
}

func findNode(root *cdp.Node, matcher func(node *cdp.Node) (matches bool)) (node *cdp.Node, found bool) {
	walkNodes(node, func(n *cdp.Node) (keepGoing bool) {
		if matcher(node) {
			node = n
			found = true
			return false
		}
		return true
	})
	return
}

func hasText(text string) func(*cdp.Node) bool {
	return func(node *cdp.Node) bool {
		_, found := findNode(node, func(n *cdp.Node) bool {
			b := n.NodeName == "#text" && n.NodeValue == text
			fmt.Printf("%v node %s=%q\n", n.PartialXPath(), n.NodeName, n.NodeValue)
			return b
		})
		return found
	}
}

func filterNodes(nodes *[]*cdp.Node, matches func(*cdp.Node) bool) chromedp.ActionFunc {
	return func(context.Context) error {
		filteredNodes := make([]*cdp.Node, 0, len(*nodes))
		for _, node := range *nodes {
			if matches(node) {
				fmt.Printf("MATCH: %s=%q\n", node.NodeName, node.NodeValue)
				filteredNodes = append(filteredNodes, node)
			} else {
				walkNodes(node, func(node *cdp.Node) bool {
					fmt.Printf("no match: %s=%q attrs=%v %v value=%q\n", node.NodeName, node.NodeValue, node.Attributes, node.Children, node.Value)
					return true
				})
			}
		}
		*nodes = filteredNodes
		return nil
	}
}
