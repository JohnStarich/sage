package prompter

import (
	"context"

	"github.com/pkg/errors"
)

type Prompter interface {
	// PromptChoice synchronously prompts user to choose an option from a list
	PromptChoice(ctx context.Context, message string, choices []string) (int, error)
	// PromptText synchronously prompts a user to enter some text
	PromptText(ctx context.Context, message string) (string, error)
	// Requests returns a channel to listen for prompt requests
	Requests() <-chan Request
	// Respond submits a response to a previously issued prompt
	Respond(resp Response)
}

type prompt struct {
	requests  chan Request
	responses chan Response
}

func New() Prompter {
	return &prompt{
		requests:  make(chan Request),
		responses: make(chan Response, 1),
	}
}

func (p *prompt) Respond(resp Response) {
	if cap(p.responses) > 0 {
		p.responses <- resp
	}
}

func (p *prompt) Requests() <-chan Request {
	return p.requests
}

func (p *prompt) PromptChoice(ctx context.Context, message string, choices []string) (int, error) {
	p.requests <- newChoicesRequest(message, choices)
	select {
	case response := <-p.responses:
		if response.Err != nil {
			return 0, response.Err
		}
		if response.Choice < 0 || response.Choice >= len(choices) {
			return 0, errors.Errorf("Invalid choice #: %d", response.Choice)
		}
		return response.Choice, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (p *prompt) PromptText(ctx context.Context, message string) (string, error) {
	p.requests <- newTextRequest(message)
	select {
	case response := <-p.responses:
		return response.Text, response.Err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
