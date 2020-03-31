package drivers

import (
	"context"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

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
						return chromedp.SendKeys(selector, id).Do(ctx)
					}
				}
			}
			return errors.New("No option selected")
		}),
	}
}
