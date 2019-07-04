package client

import "encoding/json"

// Password is a string that makes it hard to expose the underlying string outside this package
// This isn't a 100% fool-proof way to omit passwords, but it does require explicit casting to marshal the real value
type Password struct {
	*string
}

// NewPassword returns a password set to s
func NewPassword(s string) Password {
	return Password{&s}
}

// UnmarshalJSON deserializes b into a password
func (p *Password) UnmarshalJSON(b []byte) error {
	var s string
	p.string = &s
	return json.Unmarshal(b, &s)
}

// MarshalJSON returns JSON 'null' to prevent serialization within a struct
func (p Password) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// Set changes the password value to newPassword
func (p Password) Set(newPassword Password) {
	*p.string = *newPassword.string
}

// IsEmpty returns true if the internal password string is not set
func (p Password) IsEmpty() bool {
	return p.string == nil || *p.string == ""
}
