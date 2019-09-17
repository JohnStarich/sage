package redactor

import (
	"encoding/json"
	"io"
	"runtime"
)

// String is redacted when marshaling unless using redactor.Encoder
type String string

// MarshalJSON implements json.Marshaler
func (s String) MarshalJSON() ([]byte, error) {
	if isRedacted() {
		return []byte("null"), nil
	}
	return json.Marshal(string(s))
}

// Encoder marshals values into JSON with redacted values included. Only use this when persisting to disk and NOT sending over HTTP.
type Encoder json.Encoder

// NewEncoder creates a new json.Encoder
func NewEncoder(w io.Writer) *Encoder {
	return (*Encoder)(json.NewEncoder(w))
}

func (p *Encoder) toJSONEncoder() *json.Encoder {
	return (*json.Encoder)(p)
}

// Encode calls json.Encoder.Encode
func (p *Encoder) Encode(v interface{}) error {
	return p.toJSONEncoder().Encode(v)
}

// SetIndent calls json.Encoder.SetIndent
func (p *Encoder) SetIndent(prefix, indent string) {
	p.toJSONEncoder().SetIndent(prefix, indent)
}

// SetEscapeHTML calls json.Encoder.SetEscapeHTML
func (p *Encoder) SetEscapeHTML(on bool) {
	p.toJSONEncoder().SetEscapeHTML(on)
}

func isRedacted() bool {
	// poor man's redactor -- yes it's terrible, no I couldn't think of a better way
	var pc uintptr
	ok := true
	for caller := 0; ok; caller++ { // start by skipping the current function
		pc, _, _, ok = runtime.Caller(caller)
		if ok && runtime.FuncForPC(pc).Name() == "github.com/johnstarich/sage/redactor.(*Encoder).Encode" {
			return false
		}
	}
	return true
}
