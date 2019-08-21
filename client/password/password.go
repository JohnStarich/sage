package password

import "encoding/json"

// Marshaler includes a password in it's marshaled output of MarshalWithPassword
type Marshaler interface {
	MarshalWithPassword() ([]byte, error)
}

// Password is a string that makes it hard to expose the underlying string outside this package
// Only JSON-serializes to 'null'
type Password struct {
	password *string
}

// New returns a password set to s
func New(s string) *Password {
	return &Password{password: &s}
}

// UnmarshalJSON deserializes b into a password
func (p *Password) UnmarshalJSON(b []byte) error {
	var s string
	p.password = &s
	return json.Unmarshal(b, &s)
}

// MarshalJSON returns JSON 'null' to prevent serialization within a struct
func (p *Password) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// Set changes the password value to newPassword
func (p *Password) Set(newPassword *Password) {
	if newPassword.password == nil {
		return
	}
	if p.password == nil {
		p.password = new(string)
	}
	*p.password = *newPassword.password
}

// IsEmpty returns true if the internal password string is not set
func (p *Password) IsEmpty() bool {
	return p.password == nil || *p.password == ""
}

// passwordString must not be exported. Returns the real password as a string.
func (p *Password) PasswordString() string {
	if p.password == nil {
		return ""
	}
	return *p.password
}
