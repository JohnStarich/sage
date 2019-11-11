package drivers

import (
	"bytes"
	"context"
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/johnstarich/sage/client/web"
	"github.com/pkg/errors"
)

func init() {
	web.Register((&connectorDiscoverCard{}).Description(), func(connector web.CredConnector) (web.Connector, error) {
		p, ok := connector.(web.PasswordConnector)
		if !ok {
			return nil, errors.Errorf("Unsupported connector: %T %+v", connector, connector)
		}
		return &connectorDiscoverCard{p}, nil
	})
}

type connectorDiscoverCard struct {
	web.PasswordConnector
}

func (c *connectorDiscoverCard) Description() string {
	return "Discover Card"
}

func (c *connectorDiscoverCard) FID() string {
	return "9625"
}

func (c *connectorDiscoverCard) Org() string {
	return "Discover Card Account Center"
}

func (c *connectorDiscoverCard) Statement(browser web.Browser, start, end time.Time, accountID string) (*ofxgo.Response, error) {
	ctx := context.Background() // TODO add timeouts

	err := browser.Run(ctx,
		network.ClearBrowserCookies(),

		// credit card login
		chromedp.ActionFunc(func(ctx context.Context) error {
			// regular chromedp.Navigate fails if a main script (libdiscover.js) has an uncaught exception
			_, _, _, err := page.Navigate("https://portal.discover.com/customersvcs/universalLogin/ac_main").Do(ctx)
			return err
		}),
		chromedp.WaitReady(`#login-form-content`),
		chromedp.Click(`#userid-content`),
		chromedp.SetValue(`#userid-content`, c.Username()),
		chromedp.Click(`#password-content`),
		chromedp.SetValue(`#password-content`, string(c.Password())),
		chromedp.Submit(`#login-form-content`),
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

	err = browser.Run(ctx,
		// navigate to statements page
		chromedp.WaitReady(`document`),
		chromedp.Click(`.navbar-links .parent-link`),
		chromedp.Click(`a[href="/cardmembersvcs/statements/app/ctd"]`),
		// go to 12 month history
		chromedp.Click(`.activity-period-title`),
		chromedp.Click(`a[href="?view=R#/ytd"]`),
		// start OFX / QFX download
		chromedp.Click(`#download`),
		chromedp.Click(`input[value="quicken"]`),
		chromedp.Click(`#submitDownload`),
	)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case file := <-downloadedFiles:
		return ofxgo.ParseResponse(bytes.NewReader(file))
	}
}
