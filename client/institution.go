package client

type Institution interface {
	Description() string
	FID() string
	Org() string
	URL() string
	Username() string
	Password() string

	Config() Config
}

type institution struct {
	description string
	fid         string
	org         string
	url         string
	username    string
	password    string

	config Config
}

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
		password:    password,
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

func (i institution) Password() string {
	return i.password
}

func (i institution) Description() string {
	return i.description
}

func (i institution) Config() Config {
	return i.config
}
