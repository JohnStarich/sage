package records

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Record contains data that can be attached to errors for more context. i.e. screen recording gif
type Record interface {
	// ContentType returns the MIME type for this record's data. e.g. image/gif, text/plain
	ContentType() string
	// Data returns the record data in bytes
	Data() []byte
}

// Error is an error with attached records
type Error interface {
	error
	Records() []Record
}

type record struct {
	createdTime time.Time
	contentType string
	data        []byte

	dataFn   func() (contentType string, data []byte)
	dataOnce sync.Once
}

func (r *record) CreatedTime() time.Time {
	return r.createdTime
}

func (r *record) ensureData() {
	r.dataOnce.Do(func() {
		r.contentType, r.data = r.dataFn()
	})
}

func (r *record) ContentType() string {
	r.ensureData()
	return r.contentType
}

func (r *record) Data() []byte {
	r.ensureData()
	return r.data
}

func (r *record) MarshalJSON() ([]byte, error) {
	r.ensureData()
	return json.Marshal(struct {
		CreatedTime time.Time `json:",omitempty"`
		ContentType string
		Data        []byte `json:",omitempty"`
	}{
		CreatedTime: r.createdTime,
		ContentType: r.contentType,
		Data:        r.data,
	})
}

type errRecords struct {
	error
	records []Record
}

// WrapError wraps 'err' with additional records
func WrapError(err error, records ...Record) Error {
	if err == nil {
		// follow behavior of errors.Wrap
		return nil
	}
	return &errRecords{
		error:   err,
		records: records,
	}
}

func (e *errRecords) Error() string {
	return fmt.Sprintf("Records captured [%d]: %s", len(e.records), e.error.Error())
}

func (e *errRecords) AddRecord(r Record) {
	e.records = append(e.records, r)
}

func (e *errRecords) Records() []Record {
	return e.records
}

func (e *errRecords) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error   error
		Records []Record
	}{
		Error:   e.error,
		Records: e.records,
	})
}
