package client

import "encoding/json"

// Institution represents the connection and identification details for a financial institution
type Institution interface {
	Description() string
	FID() string
	Org() string
	URL() string
	Username() string
	Password() Password

	Config() Config
}

type institution struct {
	description string
	fid         string
	org         string
	url         string
	username    string
	password    Password

	config Config
}

// NewInstitution creates an institution
func NewInstitution(
	description,
	fid,
	org,
	url,
	username, password string,
	config Config,
) Institution {
	return institution{
		config:      config,
		description: description,
		fid:         fid,
		org:         org,
		password:    Password(password),
		url:         url,
		username:    username,
	}
}

func (i institution) URL() string {
	return i.url
}

func (i institution) Org() string {
	return i.org
}

func (i institution) FID() string {
	return i.fid
}

func (i institution) Username() string {
	return i.username
}

func (i institution) Password() Password {
	return i.password
}

func (i institution) Description() string {
	return i.description
}

func (i institution) Config() Config {
	return i.config
}

func (i institution) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Description string
		FID         string
		Org         string
		URL         string
		Username    string
		Config
	}{
		i.description,
		i.fid,
		i.org,
		i.url,
		i.username,
		i.config,
	})
}
