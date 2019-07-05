package client

import "encoding/json"

// Institution represents the connection and identification details for a financial institution
type Institution interface {
	Description() string
	FID() string
	Org() string
	URL() string
	Username() string
	Password() *Password

	Config() Config
}

type baseInstitution struct {
	description string
	fid         string
	org         string
	url         string
	username    string
	password    *Password

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
	return baseInstitution{
		config:      config,
		description: description,
		fid:         fid,
		org:         org,
		password:    NewPassword(password),
		url:         url,
		username:    username,
	}
}

func newBaseFromInterface(inst Institution) baseInstitution {
	var pass *Password
	if interfacePass := inst.Password(); interfacePass != nil {
		pass = NewPassword(interfacePass.passwordString())
	}
	return baseInstitution{
		config:      inst.Config(),
		description: inst.Description(),
		fid:         inst.FID(),
		org:         inst.Org(),
		password:    pass,
		url:         inst.URL(),
		username:    inst.Username(),
	}
}

func (i baseInstitution) URL() string {
	return i.url
}

func (i baseInstitution) Org() string {
	return i.org
}

func (i baseInstitution) FID() string {
	return i.fid
}

func (i baseInstitution) Username() string {
	return i.username
}

func (i baseInstitution) Password() *Password {
	return i.password
}

func (i baseInstitution) Description() string {
	return i.description
}

func (i baseInstitution) Config() Config {
	return i.config
}

type baseInstitutionJSON struct {
	Description string
	FID         string
	Org         string
	URL         string
	Username    string
	Password    string
	Config
}

func (i *baseInstitution) UnmarshalJSON(b []byte) error {
	var inst baseInstitutionJSON
	if err := json.Unmarshal(b, &inst); err != nil {
		return err
	}
	i.description = inst.Description
	i.fid = inst.FID
	i.org = inst.Org
	i.url = inst.URL
	i.username = inst.Username
	i.password = NewPassword(inst.Password)
	i.config = inst.Config
	return nil
}

func (i baseInstitution) prepMarshal() baseInstitutionJSON {
	return baseInstitutionJSON{
		Description: i.description,
		FID:         i.fid,
		Org:         i.org,
		URL:         i.url,
		Username:    i.username,
		Config:      i.config,
		// no password
	}
}

func (i baseInstitution) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.prepMarshal())
}

func (i baseInstitution) MarshalWithPassword() ([]byte, error) {
	data := i.prepMarshal()
	data.Password = i.password.passwordString()
	return json.Marshal(data)
}
