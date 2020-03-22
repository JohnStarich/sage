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
	"github.com/johnstarich/sage/client/web"
	"github.com/pkg/errors"
)

func init() {
	web.Register((&connectorUFCU{}).Description(), func(connector web.CredConnector) (web.Connector, error) {
		p, ok := connector.(web.PasswordConnector)
		if !ok {
			return nil, errors.Errorf("Unsupported connector: %T %+v", connector, connector)
		}
		return &connectorUFCU{p}, nil
	})
}

type connectorUFCU struct {
	web.PasswordConnector
}

func (c *connectorUFCU) Description() string {
	return "University Federal Credit Union"
}

func (c *connectorUFCU) FID() string {
	return "9946"
}

func (c *connectorUFCU) Org() string {
	return "UFCU"
}

func (c *connectorUFCU) Statement(browser web.Browser, start, end time.Time, accountID string) (*ofxgo.Response, error) {
	hyphenIndex := strings.IndexByte(accountID, '-')
	if hyphenIndex == -1 {
		return nil, errors.New("Invalid UFCU account ID: Must contain the bank account number followed by the share number, e.g. 123456-S0000")
	}
	share := accountID[hyphenIndex+1:]

	ctx := context.Background()

	err := browser.Run(ctx,
		network.ClearBrowserCookies(),

		// login
		chromedp.ActionFunc(func(ctx context.Context) error {
			// regular chromedp.Navigate fails if a main script has an uncaught exception
			_, _, _, err := page.Navigate("https://www.ufcu.org").Do(ctx)
			return err
		}),
		chromedp.WaitReady(`document`),
		chromedp.WaitVisible(`#mainForm`),
		chromedp.Click(`#mainForm #ctlUserName`),
		chromedp.SetValue(`#mainForm #ctlUserName`, c.Username()),
		chromedp.Click(`#mainForm #txtPassword`),
		chromedp.SetValue(`#mainForm #txtPassword`, string(c.Password())),
		chromedp.Click(`#mainForm input[value="Log In"]`),
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

	const dateFormat = "01/02/2006"
	err = browser.Run(ctx,
		chromedp.WaitReady(`document`),
		chromedp.WaitVisible(`a[aria-label="History Export"]`, chromedp.ByQuery),
		chromedp.Click(`a[aria-label="History Export"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`#qtip-someid-content img[alt="Quicken"]`),
		chromedp.Click(`#qtip-someid-content img[alt="Quicken"]`),
		chromedp.WaitVisible(`#shrd_hecrd_srch_ah_accountsInput`, chromedp.ByID),
		selectOption(`#shrd_hecrd_srch_ah_accountsInput`, func(ctx context.Context, id, content string) bool {
			return strings.Contains(content, share)
		}),
		// this piece works even though the UI elements do not update, the download URL is correct
		chromedp.SetValue(`#shrd_hecrd_mainForm #srch_ah_searchCriteriaInput`, "DateRange"),
		// enter start and end dates
		chromedp.SetValue(`#shrd_hecrd_mainForm #shrd_hecrd_srch_ah_filterStartDateInput_datepicker`, start.Format(dateFormat)),
		chromedp.SetValue(`#shrd_hecrd_mainForm #shrd_hecrd_srch_ah_filterEndDateInput_datepicker`, end.Format(dateFormat)),
		/*
			// Alternatively, a click to focus and type the option name to select also works:
			chromedp.WaitReady(`#shrd_hecrd_mainForm #srch_ah_searchCriteriaInput`),
			chromedp.Click(`#shrd_hecrd_mainForm #srch_ah_searchCriteriaInput`),
			chromedp.Click(`#shrd_hecrd_mainForm #srch_ah_searchCriteriaInput`),
			chromedp.KeyEvent("Start Date"),
		*/

		// start OFX / QFX download
		chromedp.Click(`#btn_hec_submit`),
	)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case file := <-downloadedFiles:
		resp, err := ofxgo.ParseResponse(bytes.NewReader(file))
		return resp, errors.Wrap(err, "Failed to parse response")
	}
}

func selectOption(
	selector interface{},
	shouldSelect func(ctx context.Context, id, content string) bool,
	opts ...chromedp.QueryOption,
) chromedp.Tasks {
	var selectNodes []*cdp.Node
	return []chromedp.Action{
		chromedp.Nodes(selector, &selectNodes, opts...),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var allOptionNodes []*cdp.Node
			for _, selectNode := range selectNodes {
				allOptionNodes = append(allOptionNodes, selectNode.Children...)
			}
			if len(allOptionNodes) == 0 {
				return errors.Errorf("No options matched selector: %q", selector)
			}

			for _, option := range allOptionNodes {
				if option.NodeName == "OPTION" && len(option.Children) >= 1 {
					id := option.AttributeValue("value")
					content := option.Children[0].NodeValue
					if shouldSelect(ctx, id, content) {
						return chromedp.SetValue(selector, id).Do(ctx)
					}
				}
			}
			return errors.New("No option selected")
		}),
	}
}
