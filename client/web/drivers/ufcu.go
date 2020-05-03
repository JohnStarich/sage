package drivers

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/johnstarich/sage/client/web"
	"github.com/johnstarich/sage/prompter"
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

func (c *connectorUFCU) Validate(accountID string) error {
	_, err := ufcuShareFromAccountID(accountID)
	return err
}

func ufcuShareFromAccountID(accountID string) (string, error) {
	hyphenIndex := strings.IndexByte(accountID, '-')
	if hyphenIndex == -1 {
		return "", errors.New("Invalid UFCU account ID: Must contain the bank account number followed by the share number, e.g. 123456-S0000")
	}
	return accountID[hyphenIndex+1:], nil
}

func (c *connectorUFCU) Statement(start, end time.Time, accountID string, browser web.Browser, _ prompter.Prompter) (*ofxgo.Response, error) {
	share, err := ufcuShareFromAccountID(accountID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	err = browser.Run(ctx,
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
