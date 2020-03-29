package drivers

import (
	"bytes"
	"context"
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
			Logger:            logger,
		}, err
	})
}

type connectorAlly struct {
	web.PasswordConnector

	Logger *zap.Logger
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

func (c *connectorAlly) Statement(browser web.Browser, start, end time.Time, accountID string) (*ofxgo.Response, error) {
	const maxStatementFetchTime = 2 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), maxStatementFetchTime)
	defer cancel()
	start = start.Local()
	end = end.Local()

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
		chromedp.SendKeys(`#login-password`, string(c.Password())+kb.Enter),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to log in")
	}

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
		return nil, errors.Wrap(err, "Failed to prepare statement download request")
	}

	{
		var submitErr error
		for ; start.Unix() < end.Unix(); start = start.AddDate(0, 0, 1) {
			// if there's an issue with the start date, try later dates
			var errorNodes []*cdp.Node
			submitErr = browser.Run(ctx,
				chromedp.SetValue(`#downloadStartDate`, ""),
				chromedp.SendKeys(`#downloadStartDate`, start.Format(dateFormat)),
				chromedp.Click(`.transactions-history button[type="submit"]`),
				chromedp.Sleep(100*time.Millisecond),
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
			)
			if submitErr == nil || !strings.Contains(submitErr.Error(), "Please enter a start date that falls after your account open date.") {
				break
			}
		}
		if submitErr != nil {
			return nil, errors.Wrap(submitErr, "Failed to submit history download request")
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
			c.Logger.Warn("OFX response failed validation", zap.Error(err))
			err = nil
		}
		return resp, errors.Wrap(err, "Failed to parse response")
	}
}
