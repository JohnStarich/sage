package web

import (
	"context"

	"github.com/johnstarich/sage/records"
)

type browserRecorder struct {
	Browser
	recorder records.ScreenRecorder
}

func (b *browserRecorder) Run(ctx context.Context, actions ...Action) error {
	if b.recorder == nil {
		b.recorder = records.NewScreenRecorder(1.25)
	}

	for i, action := range actions {
		actions[i] = &actionRecorder{recorder: b.recorder, Action: action}
	}

	runErr := b.Browser.Run(ctx, actions...)
	return records.WrapError(runErr, b.recorder.Encode())
}

type actionRecorder struct {
	recorder records.ScreenRecorder
	Action
}

func (r *actionRecorder) Do(ctx context.Context) error {
	err := r.Action.Do(ctx)
	_ = r.recorder.Capture(ctx)
	return err
}
