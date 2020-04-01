package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color/palette"
	"image/gif"
	"image/png"
	"io"
	"time"

	"github.com/chromedp/cdproto/page"
	"golang.org/x/image/draw"
)

type Recorder interface {
	Snapshot(ctx context.Context) error
	Encode() (Record, error)
}

type Record interface {
	ContentType() string
	Data() []byte
}

type ErrWithRecordings interface {
	error
	Recordings() []Record
}

func NewRecorder(timeScale float64) Recorder {
	return &screenRecorder{
		timeScale: timeScale,
	}
}

type screenRecorder struct {
	captures  []frame
	timeScale float64
}

type frame struct {
	time time.Time
	data []byte
}

func (s *screenRecorder) Snapshot(ctx context.Context) error {
	now := time.Now()
	buf, err := (&page.CaptureScreenshotParams{
		Format:      page.CaptureScreenshotFormatPng,
		Quality:     0,
		FromSurface: true,
	}).Do(ctx)
	if err != nil {
		return err
	}
	s.captures = append(s.captures, frame{data: buf, time: now})
	return nil
}

func (s *screenRecorder) Encode() (Record, error) {
	data, err := makeGifWithDecoder(s.timeScale, png.Decode, s.captures...)
	return &record{data: data, contentType: "image/gif"}, err
}

type record struct {
	contentType string
	data        []byte
}

func (r *record) ContentType() string {
	return r.contentType
}

func (r *record) Data() []byte {
	return r.data
}

func (r *record) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ContentType string
		Data        []byte
	}{
		ContentType: r.contentType,
		Data:        r.data,
	})
}

func makeGifWithDecoder(timeScale float64, decoder func(io.Reader) (image.Image, error), frames ...frame) ([]byte, error) {
	result := &gif.GIF{}

	for ix, f := range frames {
		delay := 5 * time.Second // pause at end of gif (default)
		if ix < len(frames)-1 {
			delay = frames[ix+1].time.Sub(f.time)
		}
		result.Delay = append(result.Delay, int(timeScale*float64(delay.Nanoseconds()/1e7))) // hundredths of a second

		img, err := decoder(bytes.NewReader(f.data))
		if err != nil {
			return nil, err
		}
		palletteImg := image.NewPaletted(img.Bounds(), palette.WebSafe)
		draw.Draw(palletteImg, palletteImg.Rect, img, img.Bounds().Min, draw.Over)
		result.Image = append(result.Image, palletteImg)
	}
	var buf bytes.Buffer
	err := gif.EncodeAll(&buf, result)
	return buf.Bytes(), err
}

type errRecordings struct {
	error
	recordings []Record
}

func WrapErrWithRecordings(err error, recordings ...Record) ErrWithRecordings {
	if err == nil {
		// follow behavior of errors.Wrap
		return nil
	}
	return &errRecordings{
		error:      err,
		recordings: recordings,
	}
}

func (e *errRecordings) Error() string {
	return fmt.Sprintf("Recordings captured [%d]: %s", len(e.recordings), e.error.Error())
}

func (e *errRecordings) AddRecording(r Record) {
	e.recordings = append(e.recordings, r)
}

func (e *errRecordings) Recordings() []Record {
	return e.recordings
}

func (e *errRecordings) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error      error
		Recordings []Record
	}{
		Error:      e.error,
		Recordings: e.recordings,
	})
}
