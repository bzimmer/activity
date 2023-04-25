package activity_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
)

type upload struct {
	done bool
}

func (u *upload) Identifier() activity.UploadID {
	return activity.UploadID(1122)
}

func (u *upload) Done() bool {
	return u.done
}

type uploader struct {
	err               bool
	status, statuscnt int
}

func (u *uploader) Upload(_ context.Context, _ *activity.File) (activity.Upload, error) {
	return &upload{}, nil
}

func (u *uploader) Status(_ context.Context, _ activity.UploadID) (activity.Upload, error) {
	defer func() { u.statuscnt++ }()
	if u.err {
		return nil, errors.New("uploader error")
	}
	return &upload{done: u.status == u.statuscnt}, nil
}

func TestPoller(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name       string
		err        bool
		exceeds    bool
		it, status int
		in, to     time.Duration
	}{
		{name: "< iterations", status: 3, it: 5, in: time.Millisecond * 10},
		{name: "max iterations", status: 100, it: 5, in: time.Millisecond * 10, exceeds: true},
		{name: "errors", status: 1, it: 5, in: time.Millisecond * 10, err: true},
		{name: "ctx timeout", status: 1, it: 5, in: time.Second, to: time.Millisecond * 10, err: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if tt.to > 0 {
				var cancel func()
				ctx, cancel = context.WithTimeout(ctx, tt.to)
				defer cancel()
			}
			u := &uploader{status: tt.status, err: tt.err}
			p := activity.NewPoller(u, activity.WithInterval(tt.in), activity.WithIterations(tt.it))
			a.NotNil(p)
			for x := range p.Poll(ctx, activity.UploadID(11011)) {
				a.NotNil(x)
				if x.Err != nil && tt.exceeds {
					a.ErrorIs(x.Err, activity.ErrExceededIterations)
					continue
				}
				switch tt.err {
				case true:
					a.Nil(x.Upload)
					a.Error(x.Err)
				case false:
					a.NoError(x.Err)
					a.NotNil(x.Upload)
				}
			}
		})
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	a.Equal("fit", activity.FormatFIT.String())

	a.Equal(activity.FormatFIT, activity.ToFormat("fit"))
	a.Equal(activity.FormatTCX, activity.ToFormat("tcx"))
	a.Equal(activity.FormatGPX, activity.ToFormat("gpx"))
	a.Equal(activity.FormatOriginal, activity.ToFormat(""))

	v, err := json.Marshal(activity.FormatFIT)
	a.NoError(err)
	a.JSONEq(`"fit"`, string(v))
}

func TestFile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	r := strings.NewReader("foo")
	f := activity.File{
		Reader: r,
	}
	a.NoError(f.Close())

	f = activity.File{}
	a.NoError(f.Close())
}
