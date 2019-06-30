package client

// Password is a string that can't be marshalled into JSON
// This isn't a 100% fool-proof way to omit passwords, but it does require explicit casting to marshal the real value
type Password string

func (p Password) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}
