package direct

// Config contains financial institution connection details
type Config struct {
	AppID      string
	AppVersion string
	ClientID   string `json:",omitempty"`
	OFXVersion string
	NoIndent   bool `json:",omitempty"`
}
