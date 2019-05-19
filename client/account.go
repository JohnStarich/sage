package client

import (
	"time"

	"github.com/aclindsa/ofxgo"
)

type Account interface {
	ID() string
	Description() string
	Institution() Institution

	Statement(time.Duration) (ofxgo.Request, error)
}

type baseAccount struct {
	id          string
	institution Institution
}

func (b baseAccount) Institution() Institution {
	return b.institution
}

func (b baseAccount) ID() string {
	return b.id
}

func (b baseAccount) Description() string {
	// TODO not implemented
	return ""
}
