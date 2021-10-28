package strava_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/bzimmer/httpwares"
	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/strava"
)

func readall(ctx context.Context, client *strava.Client, spec activity.Pagination, opts ...strava.APIOption) ([]*strava.Activity, error) {
	var activities []*strava.Activity
	err := strava.ActivitiesIter(
		client.Activity.Activities(ctx, spec, opts...),
		func(act *strava.Activity) (bool, error) {
			activities = append(activities, act)
			return true, nil
		})
	return activities, err
}

func TestActivity(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	client, err := newClient(http.StatusOK, "activity.json")
	a.NoError(err)
	ctx := context.Background()
	act, err := client.Activity.Activity(ctx, 154504250376823)
	a.NoError(err)
	a.NotNil(act)
	a.Equal(int64(154504250376823), act.ID)
}

func TestActivities(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	ctx := context.Background()
	client, err := strava.NewClient(
		strava.WithTransport(&ManyTransport{
			Filename: "testdata/activity.json",
			Total:    2,
		}),
		strava.WithTokenCredentials("fooKey", "barToken", time.Time{}))
	a.NoError(err)

	acts, err := readall(ctx, client, activity.Pagination{})
	a.NoError(err)
	a.Equal(2, len(acts))
}

func TestActivitiesWithOptions(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tt := range []struct {
		name string
		err  bool
		opt  strava.APIOption
	}{
		{
			name: "nil options",
			opt:  nil,
		},
		{
			name: "err options",
			err:  true,
			opt: func(url.Values) error {
				return errors.New("error in option")
			},
		},
		{
			name: "zero dates",
			opt:  strava.WithDateRange(time.Time{}, time.Time{}),
		},
		{
			name: "before and after",
			opt: func() strava.APIOption {
				before := time.Now()
				after := before.Add(time.Hour * time.Duration(-24*7))
				return strava.WithDateRange(before, after)
			}(),
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := strava.NewClient(
				strava.WithTransport(&ManyTransport{
					Filename: "testdata/activity.json",
					Total:    2,
				}),
				strava.WithTokenCredentials("fooKey", "barToken", time.Time{}))
			a.NoError(err)
			acts, err := readall(ctx, client, activity.Pagination{}, tt.opt)
			if tt.err {
				a.Error(err)
			} else {
				a.NoError(err)
				a.Equal(2, len(acts))
			}
		})
	}
}

type F struct {
	n int
}

func (f *F) X(res *http.Response) error {
	if f.n == 1 {
		// on the second iteration return an empty body signaling no more activities exist
		res.ContentLength = int64(0)
		res.Body = io.NopCloser(bytes.NewBuffer([]byte{}))
	}
	f.n++
	return nil
}

func TestActivitiesRequestedGTAvailable(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, err := newClienter(http.StatusOK, "activities.json", nil, (&F{}).X)
	a.NoError(err)
	ctx := context.Background()
	acts, err := readall(ctx, client, activity.Pagination{Total: 325})
	a.NoError(err)
	a.Equal(2, len(acts))
}

func TestActivitiesMany(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	ctx := context.Background()
	client, err := strava.NewClient(
		strava.WithTransport(&ManyTransport{
			Filename: "testdata/activity.json",
		}),
		strava.WithTokenCredentials("fooKey", "barToken", time.Time{}))
	a.NoError(err)

	t.Run("total, start, and count", func(t *testing.T) {
		// success: the requested number of activities because count/pagesize == 1
		acts, err := readall(ctx, client, activity.Pagination{Total: 127, Start: 0, Count: 1})
		a.NoError(err)
		a.NotNil(acts)
		a.Equal(127, len(acts))
	})

	t.Run("total and start", func(t *testing.T) {
		// success: the requested number of activities is exceeded because count/pagesize not specified
		x := 234
		acts, err := readall(ctx, client, activity.Pagination{Total: x, Start: 0})
		a.NoError(err)
		a.NotNil(acts)
		a.Equal(x, len(acts))
	})

	t.Run("total and start less than PageSize", func(t *testing.T) {
		// success: the requested number of activities because count/pagesize <= strava.PageSize
		a.True(27 < strava.PageSize)
		acts, err := readall(ctx, client, activity.Pagination{Total: 27, Start: 0})
		a.NoError(err)
		a.NotNil(acts)
		a.Equal(27, len(acts))
	})

	t.Run("different Count values", func(t *testing.T) {
		count := strava.PageSize + 100
		for _, x := range []int{27, 350, strava.PageSize} {
			acts, err := readall(ctx, client, activity.Pagination{Total: x, Start: 0, Count: count})
			a.NoError(err)
			a.NotNil(acts)
			a.Equal(x, len(acts))
		}
	})

	t.Run("negative total", func(t *testing.T) {
		acts, err := readall(ctx, client, activity.Pagination{Total: -1})
		a.Error(err)
		a.Nil(acts)
	})
}

func TestActivityStreams(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	t.Run("four", func(t *testing.T) {
		ctx := context.Background()
		client, err := newClient(http.StatusOK, "streams_four.json")
		a.NoError(err)
		sms, err := client.Activity.Streams(ctx, 154504250376, "latlng", "altitude", "distance")
		a.NoError(err)
		a.NotNil(sms)
		a.NotNil(sms.LatLng)
		a.NotNil(sms.Elevation)
		a.NotNil(sms.Distance)
	})

	t.Run("two", func(t *testing.T) {
		ctx := context.Background()
		client, err := newClient(http.StatusOK, "streams_two.json")
		a.NoError(err)
		sms, err := client.Activity.Streams(ctx, 154504250376, "latlng", "altitude")
		a.NoError(err)
		a.NotNil(sms)
		a.NotNil(sms.LatLng)
		a.NotNil(sms.Elevation)
	})

	t.Run("invalid stream", func(t *testing.T) {
		ctx := context.Background()
		client, err := newClient(http.StatusOK, "streams_two.json")
		a.NoError(err)
		sms, err := client.Activity.Streams(ctx, 154504250376, "foo", "bar")
		a.Error(err)
		a.Nil(sms)
		a.Contains(err.Error(), "invalid stream")
	})
}

func TestActivityTimeout(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, err := strava.NewClient(
		strava.WithTokenCredentials("fooKey", "barToken", time.Time{}),
		strava.WithTransport(&httpwares.SleepingTransport{
			Duration: time.Millisecond * 30,
			Transport: &httpwares.TestDataTransport{
				Status:      http.StatusOK,
				Filename:    "activity.json",
				ContentType: "application/json",
			}}))
	a.NoError(err)
	a.NotNil(client)

	t.Run("timeout lt sleep => failure", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Microsecond)
		defer cancel()
		act, err := client.Activity.Activity(ctx, 154504250376823)
		a.Error(err)
		a.Nil(act)
	})

	t.Run("timeout gt sleep => success", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Millisecond*120)
		defer cancel()
		act, err := client.Activity.Activity(ctx, 154504250376823)
		a.NoError(err)
		a.NotNil(act)
		a.Equal(int64(154504250376823), act.ID)
	})
}

func TestStreamSets(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	svr := httptest.NewServer(http.NewServeMux())
	defer svr.Close()

	client, err := newTestClient(strava.WithBaseURL(svr.URL))
	a.NoError(err)

	s := client.Activity.StreamSets()
	a.NotNil(s)
	a.Equal(11, len(s))
}

func TestPhotos(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	newMux := func() *http.ServeMux {
		mux := http.NewServeMux()
		mux.HandleFunc("/activities/6099369285/photos", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/photos.json")
		})
		return mux
	}

	tests := []struct {
		id   int64
		name string
	}{
		{
			id:   6099369285,
			name: "query photos",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svr := httptest.NewServer(newMux())
			defer svr.Close()

			client, err := newTestClient(strava.WithBaseURL(svr.URL))
			a.NoError(err)
			photos, err := client.Activity.Photos(context.Background(), tt.id, 2048)
			a.NoError(err)
			a.NotNil(photos)
		})
	}
}

func TestUpload(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	newMux := func() *http.ServeMux {
		up := &strava.Upload{
			ID:         12345,
			IDString:   "12345",
			ExternalID: "",
			Error:      "",
			Status:     "ok",
			ActivityID: 54321,
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/uploads", func(w http.ResponseWriter, r *http.Request) {
			enc := json.NewEncoder(w)
			a.NoError(enc.Encode(up))
		})
		mux.HandleFunc("/uploads/12345", func(w http.ResponseWriter, r *http.Request) {
			enc := json.NewEncoder(w)
			a.NoError(enc.Encode(up))
		})
		return mux
	}

	tests := []struct {
		name string
		err  bool
		done bool
		file *activity.File
	}{
		{
			name: "nil file",
			err:  true,
			file: nil,
		},
		{
			name: "valid file",
			err:  false,
			done: true,
			file: &activity.File{
				Name:     "LongHike.gpx",
				Filename: "/tmp/LongHike.gpx",
				Format:   activity.FormatGPX,
				Reader: func() io.Reader {
					var w bytes.Buffer
					fp, err := os.Open("testdata/example.gpx")
					a.NoError(err)
					defer fp.Close()
					_, err = io.Copy(&w, fp)
					a.NoError(err)
					return &w
				}(),
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			svr := httptest.NewServer(newMux())
			defer svr.Close()
			client, err := newTestClient(strava.WithBaseURL(svr.URL))
			a.NoError(err)

			uploader := client.Uploader()
			upload, err := uploader.Upload(context.Background(), tt.file)
			if tt.err {
				a.Error(err)
				a.Nil(upload)
				return
			}
			a.NoError(err)
			a.NotNil(upload)
			a.Equal(tt.done, upload.Done())

			upload, err = uploader.Status(context.Background(), upload.Identifier())
			a.NoError(err)
			a.NotNil(upload)
			a.Equal(tt.done, upload.Done())
		})
	}
}

func TestExporter(t *testing.T) {
	a := assert.New(t)

	newMux := func() *http.ServeMux {
		mux := http.NewServeMux()
		mux.HandleFunc("/activities/6099369285", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/activity.json")
		})
		mux.HandleFunc("/activities/6099369285/streams/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/streams_export.json")
		})
		return mux
	}

	tests := []struct {
		id   int64
		name string
	}{
		{
			id:   6099369285,
			name: "export activity",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svr := httptest.NewServer(newMux())
			defer svr.Close()

			client, err := newTestClient(strava.WithBaseURL(svr.URL))
			a.NoError(err)
			exporter := client.Exporter()
			export, err := exporter.Export(context.Background(), tt.id)
			a.NoError(err)
			a.NotNil(export)
		})
	}
}

func TestWithDateRange(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	tests := []struct {
		name          string
		before, after time.Time
		length        int
		err           string
	}{
		{
			name: "both zero",
		},
		{
			name:   "invalid range",
			before: time.Now(),
			after:  time.Now().Add(time.Hour),
			err:    "invalid date range",
		},
		{
			name:   "valid range",
			before: time.Now(),
			after:  time.Now().Add(-time.Hour),
			length: 2,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := url.Values{}
			opt := strava.WithDateRange(tt.before, tt.after)
			err := opt(v)
			if tt.err != "" {
				a.Error(err)
				a.Contains(err.Error(), tt.err)
				return
			}
			a.NoError(opt(v))
			a.Equal(tt.length, len(v))
		})
	}
}
