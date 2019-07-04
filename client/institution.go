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
		password:    NewPassword(password),
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

type institutionJSON struct {
	Description string
	FID         string
	Org         string
	URL         string
	Username    string
	Password    Password
	Config
}

func (i *institution) UnmarshalJSON(b []byte) error {
	var inst institutionJSON
	if err := json.Unmarshal(b, &inst); err != nil {
		return err
	}
	i.description = inst.Description
	i.fid = inst.FID
	i.org = inst.Org
	i.url = inst.URL
	i.username = inst.Username
	i.password = inst.Password
	i.config = inst.Config
	return nil
}

func (i institution) MarshalJSON() ([]byte, error) {
	return json.Marshal(institutionJSON{
		Description: i.description,
		FID:         i.fid,
		Org:         i.org,
		URL:         i.url,
		Username:    i.username,
		Config:      i.config,
	})
}
