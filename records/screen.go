package records

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/ericpauley/go-quantize/quantize"
)

// ScreenRecorder takes screenshots and encodes them as a gif
type ScreenRecorder interface {
	Capture(ctx context.Context) error
	Encode() Record
}

// NewScreenRecorder creates a new ScreenRecorder and stretches the gif frame delay by 'timeScale'
func NewScreenRecorder(timeScale float64) ScreenRecorder {
	return &screenRecorder{
		timeScale: timeScale,
	}
}

type screenRecorder struct {
	frames    []screenFrame // NOTE: do not mutate frames after adding, shallow copy is used for efficient deferred encoding
	timeScale float64
}

type screenFrame struct {
	time time.Time
	data []byte
}

func (s *screenRecorder) Capture(ctx context.Context) error {
	now := time.Now()
	buf, err := (&page.CaptureScreenshotParams{
		Format:      page.CaptureScreenshotFormatPng,
		Quality:     0,
		FromSurface: true,
	}).Do(ctx)
	if err != nil {
		return err
	}
	s.frames = append(s.frames, screenFrame{data: buf, time: now})
	return nil
}

func (s *screenRecorder) Encode() Record {
	var createdTime time.Time
	if len(s.frames) > 0 {
		createdTime = s.frames[0].time
	}
	snapshot := *s // freeze this screen recorder with the current frames
	return &record{
		createdTime: createdTime,
		dataFn:      snapshot.encode,
	}
}

func (s *screenRecorder) encode() (string, []byte) {
	data, err := makeGifWithDecoder(s.timeScale, png.Decode, s.frames...)
	if err != nil {
		return "text/plain", []byte(err.Error())
	}
	return "image/gif", data
}

func makeGifWithDecoder(timeScale float64, decoder func(io.Reader) (image.Image, error), frames ...screenFrame) ([]byte, error) {
	result := &gif.GIF{
		Delay: make([]int, 0, len(frames)),
		Image: make([]*image.Paletted, 0, len(frames)),
	}
	// use quantizer to create a palette -- much faster to Draw with than palette.WebSafe
	const maxColors = 16 // maximum number of colors in a palette, can heavily influence performance
	quantizer := (&quantize.MedianCutQuantizer{}).Quantize

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
		palette := quantizer(make([]color.Color, 0, maxColors), img)
		paletteImg := image.NewPaletted(img.Bounds(), palette)
		draw.Draw(paletteImg, paletteImg.Rect, img, img.Bounds().Min, draw.Src)
		result.Image = append(result.Image, paletteImg)
	}
	var buf bytes.Buffer
	err := gif.EncodeAll(&buf, result)
	return buf.Bytes(), err
}
