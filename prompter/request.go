package prompter

type Request struct {
	Message string
	Text    bool
	Choices []string
}

func newTextRequest(message string) Request {
	return Request{
		Message: message,
		Text:    true,
	}
}

func newChoicesRequest(message string, choices []string) Request {
	return Request{
		Message: message,
		Choices: choices,
	}
}
