package web

import (
	"context"
	"net/http"
	"sync"
	"time"

	cdpBrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Action performs a browser action, like clicking on a page or emulating a device
type Action chromedp.Action

// Downloader receives a download request and processes it synchronously
type Downloader func(context.Context, DownloadRequest) error

// Browser manages a browser window, runs actions against it, and handles download requests
type Browser interface {
	// Run calls each action in order, stopping on the first error
	Run(ctx context.Context, actions ...Action) error
	// Download registers an asynchronous handler for any detected file downloads.
	// Ensure this is registered before Run is called.
	Download(downloader Downloader)
	// DownloadErr returns an error if one has occurred during a download. Always returns the first error.
	DownloadErr() error
}

type browser struct {
	Context    context.Context
	CancelFunc context.CancelFunc

	downloadErr     error
	downloadErrOnce sync.Once
	downloadErrs    <-chan error
	downloaders     []Downloader
}

// BrowserConfig passes options for a new browser constructor
type BrowserConfig struct {
	Debug      bool
	NoHeadless bool
	Device     chromedp.Device
	Logger     *zap.Logger
}

// NewBrowser creates an instance of Browser with the given config options, bound to the provided context
// Canceling the context closes the browser entirely
func NewBrowser(ctx context.Context, config BrowserConfig) (Browser, error) {
	execOpts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
	)
	if config.NoHeadless {
		execOpts = append(
			// skip headless option
			chromedp.DefaultExecAllocatorOptions[3:],

			chromedp.NoFirstRun,
			chromedp.NoDefaultBrowserCheck,
			chromedp.DisableGPU,
		)
	}
	if config.Device == nil {
		config.Device = desktopDevice
	}
	if config.Logger == nil {
		var err error
		config.Logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := chromedp.NewExecAllocator(ctx, execOpts...)
	var ctxOpts []chromedp.ContextOption
	if config.Debug {
		logger := config.Logger.Sugar()
		ctxOpts = append(ctxOpts,
			chromedp.WithDebugf(logger.Debugf),
			chromedp.WithLogf(logger.Infof),
			chromedp.WithErrorf(logger.Errorf),
		)
	}
	ctx, _ = chromedp.NewContext(ctx, ctxOpts...)

	// set some sane defaults for all drivers
	err := chromedp.Run(ctx,
		chromedp.Emulate(config.Device),
		page.SetDownloadBehavior(page.SetDownloadBehaviorBehaviorDeny), // deny downloads so Download() can hijack them
	)
	if err != nil {
		return nil, err
	}

	return &browser{
		Context:      ctx,
		CancelFunc:   cancel,
		downloadErrs: make(chan error, 1),
	}, nil
}

func chromiumSupport() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := NewBrowser(ctx, BrowserConfig{Logger: zap.NewNop()})
	return err == nil
}

func (b *browser) Run(ctx context.Context, actions ...Action) error {
	done := ctx.Done()
	chromedpCtx := chromedp.FromContext(b.Context)
	wrapperCtx, wrapperCancel := chromedp.NewContext(cdp.WithExecutor(b.Context, chromedpCtx.Target))
	defer wrapperCancel()
	for i, action := range actions {
		select {
		case <-done:
			return ctx.Err()
		default:
		}
		err := action.Do(wrapperCtx)
		if err != nil {
			return errors.Wrapf(err, "Error running action %T with index #%d: %+v", action, i, action)
		}
	}
	return nil
}

func (b *browser) Download(downloader Downloader) {
	if len(b.downloaders) == 0 {
		downloadURLs := make(chan string)
		b.downloadErrs = runDownloader(b.Context, b.CancelFunc, downloadURLs,
			func(ctx context.Context, download DownloadRequest) error {
				for _, downloader := range b.downloaders {
					if err := downloader(ctx, download); err != nil {
						return err
					}
				}
				return nil
			})
		chromedp.ListenTarget(b.Context, func(ev interface{}) {
			if downloadEvent, ok := ev.(*page.EventDownloadWillBegin); ok {
				downloadURLs <- downloadEvent.URL
			}
		})
	}
	b.downloaders = append(b.downloaders, downloader)
}

func runDownloader(ctx context.Context, cancel context.CancelFunc, downloadURLs <-chan string, downloader Downloader) <-chan error {
	errs := make(chan error, 1)
	go func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			case url := <-downloadURLs:
				var networkCookies []*network.Cookie
				var userAgent string
				err := chromedp.Run(ctx,
					chromedp.ActionFunc(func(ctx context.Context) error {
						var err error
						networkCookies, err = network.GetCookies().WithUrls([]string{url}).Do(ctx)
						return err
					}),
					chromedp.ActionFunc(func(ctx context.Context) error {
						var err error
						_, _, _, userAgent, _, err = cdpBrowser.GetVersion().Do(ctx)
						return err
					}),
				)
				if err != nil {
					errs <- err
					return
				}
				cookies := make([]*http.Cookie, 0, len(networkCookies))
				for _, cookie := range networkCookies {
					cookies = append(cookies, &http.Cookie{
						Name:  cookie.Name,
						Value: cookie.Value,
					})
				}
				download := DownloadRequest{
					URL:       url,
					Cookies:   cookies,
					UserAgent: userAgent,
				}
				if err := downloader(ctx, download); err != nil {
					errs <- err
					return
				}
			}
		}
	}()
	return errs
}

func (b *browser) DownloadErr() error {
	if b.downloadErr == nil && cap(b.downloadErrs) == len(b.downloadErrs) {
		return nil
	}

	b.downloadErrOnce.Do(func() {
		b.downloadErr = <-b.downloadErrs
	})
	return b.downloadErr
}
