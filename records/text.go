package records

import "time"

func New(plainText string) Record {
	return &record{
		createdTime: time.Now(),
		contentType: "text/plain",
		data:        []byte(plainText),
	}
}
