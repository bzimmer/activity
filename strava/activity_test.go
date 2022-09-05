package strava_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/strava"
)

func TestActivity(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tt := range []struct {
		name    string
		timeout time.Duration
		before  func(mux *http.ServeMux)
		after   func(activity *strava.Activity, err error)
	}{
		{
			name: "valid activity",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/154504250376823", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/activity.json")
				})
			},
			after: func(activity *strava.Activity, err error) {
				a.NoError(err)
				a.NotNil(activity)
			},
		},
		{
			name:    "timeout lt sleep => failure",
			timeout: time.Millisecond,
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/154504250376823", func(w http.ResponseWriter, r *http.Request) {
					select {
					case <-r.Context().Done():
						w.WriteHeader(http.StatusInternalServerError)
					case <-time.After(15 * time.Millisecond):
						http.ServeFile(w, r, "testdata/activity.json")
					}
				})
			},
			after: func(activity *strava.Activity, err error) {
				a.Error(err)
				a.Contains(err.Error(), "context deadline exceeded")
				a.Nil(activity)
			},
		},
		{
			name:    "timeout gt sleep => success",
			timeout: time.Millisecond * 120,
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/154504250376823", func(w http.ResponseWriter, r *http.Request) {
					select {
					case <-r.Context().Done():
						w.WriteHeader(http.StatusInternalServerError)
					case <-time.After(15 * time.Millisecond):
						http.ServeFile(w, r, "testdata/activity.json")
					}
				})
			},
			after: func(activity *strava.Activity, err error) {
				a.NoError(err)
				a.NotNil(activity)
			},
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			ctx := context.TODO()
			if tt.timeout > 0 {
				var cancel func()
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			}
			tt.after(client.Activity.Activity(ctx, 154504250376823))
		})
	}
}

func TestActivities(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	all := func(res <-chan *strava.ActivityResult) []*strava.Activity {
		var acts []*strava.Activity
		a.NoError(strava.ActivitiesIter(res,
			func(act *strava.Activity) (bool, error) {
				acts = append(acts, act)
				return true, nil
			}))
		return acts
	}

	tests := []struct {
		name       string
		pagination activity.Pagination
		opt        strava.APIOption
		before     func(mux *http.ServeMux)
		after      func(activities <-chan *strava.ActivityResult)
	}{
		{
			name:       "request two, get two",
			pagination: activity.Pagination{Total: 2},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Total:    2,
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				acts := all(activities)
				a.Len(acts, 2)
			},
		},
		{
			name:       "fewer activities than requested",
			pagination: activity.Pagination{Total: 325},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Total:    15,
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				acts := all(activities)
				a.Len(acts, 15)
			},
		},
		{
			name:       "total, start, and count",
			pagination: activity.Pagination{Total: 127, Start: 0, Count: 1},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				acts := all(activities)
				a.Len(acts, 127)
			},
		},
		{
			name:       "total and start",
			pagination: activity.Pagination{Total: 234, Start: 0},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				acts := all(activities)
				a.Len(acts, 234)
			},
		},
		{
			name:       "total and start less than PageSize",
			pagination: activity.Pagination{Total: 27, Start: 0},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				acts := all(activities)
				a.Len(acts, 27)
			},
		},
		{
			name:       "exact PageSize",
			pagination: activity.Pagination{Total: strava.PageSize, Count: strava.PageSize + 100},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				acts := all(activities)
				a.Len(acts, strava.PageSize)
			},
		},
		{
			name:       "zero dates",
			opt:        strava.WithDateRange(time.Time{}, time.Time{}),
			pagination: activity.Pagination{Total: 2},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Total:    2,
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				acts := all(activities)
				a.Len(acts, 2)
			},
		},
		{
			name: "before and after",
			opt: func() strava.APIOption {
				before := time.Now()
				after := before.Add(time.Hour * time.Duration(-24*7))
				return strava.WithDateRange(before, after)
			}(),
			pagination: activity.Pagination{Total: 2},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Total:    2,
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				acts := all(activities)
				a.Len(acts, 2)
			},
		},
		{
			name: "error in option",
			opt: func(url.Values) error {
				return errors.New("error in option")
			},
			pagination: activity.Pagination{Total: 2},
			before: func(mux *http.ServeMux) {
				mux.Handle("/athlete/activities", &ManyHandler{
					Total:    2,
					Filename: "testdata/activity.json",
				})
			},
			after: func(activities <-chan *strava.ActivityResult) {
				res := <-activities
				a.Error(res.Err)
				a.Nil(res.Activity)
				a.Contains(res.Err.Error(), "error in option")
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			tt.after(client.Activity.Activities(context.TODO(), tt.pagination, tt.opt))
		})
	}
}

func TestActivityStreams(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name    string
		streams []string
		before  func(mux *http.ServeMux)
		after   func(streams *strava.Streams, err error)
	}{
		{
			name:    "four",
			streams: []string{"latlng", "altitude", "distance"},
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/8002/streams/latlng,altitude,distance", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/streams_four.json")
				})
			},
			after: func(streams *strava.Streams, err error) {
				a.NoError(err)
				a.NotNil(streams)
				a.NotNil(streams.LatLng)
				a.NotNil(streams.Elevation)
				a.NotNil(streams.Distance)
			},
		},
		{
			name:    "two",
			streams: []string{"latlng", "altitude"},
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/8002/streams/latlng,altitude", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/streams_two.json")
				})
			},
			after: func(streams *strava.Streams, err error) {
				a.NoError(err)
				a.NotNil(streams)
				a.NotNil(streams.LatLng)
				a.NotNil(streams.Elevation)
			},
		},
		{
			name:    "invalid",
			streams: []string{"foo", "bar", "baz"},
			before:  func(mux *http.ServeMux) {},
			after: func(streams *strava.Streams, err error) {
				a.Error(err)
				a.Nil(streams)
				a.Contains(err.Error(), "invalid stream")
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			tt.after(client.Activity.Streams(context.TODO(), 8002, tt.streams...))
		})
	}
}

func TestStreamSets(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	client, svr := newClient(func(_ *http.ServeMux) {})
	defer svr.Close()
	s := client.Activity.StreamSets()
	a.NotNil(s)
	a.Equal(11, len(s))
}

func TestPhotos(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
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
			client, svr := newClient(func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/6099369285/photos", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/photos.json")
				})
			})
			defer svr.Close()
			photos, err := client.Activity.Photos(context.Background(), tt.id, 2048)
			a.NoError(err)
			a.NotNil(photos)
		})
	}
}

func TestUpload(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	for _, tt := range []struct {
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
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(func(mux *http.ServeMux) {
				up := &strava.Upload{
					ID:         12345,
					IDString:   "12345",
					ExternalID: "",
					Error:      "",
					Status:     "ok",
					ActivityID: 54321,
				}
				mux.HandleFunc("/uploads", func(w http.ResponseWriter, r *http.Request) {
					enc := json.NewEncoder(w)
					a.NoError(enc.Encode(up))
				})
				mux.HandleFunc("/uploads/12345", func(w http.ResponseWriter, r *http.Request) {
					enc := json.NewEncoder(w)
					a.NoError(enc.Encode(up))
				})
			})
			defer svr.Close()
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
	t.Parallel()
	a := assert.New(t)
	for _, tt := range []struct {
		id   int64
		name string
	}{
		{
			id:   6099369285,
			name: "export activity",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/6099369285", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/activity.json")
				})
				mux.HandleFunc("/activities/6099369285/streams/", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/streams_export.json")
				})
			})
			defer svr.Close()
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
	for _, tt := range []struct {
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
	} {
		tt := tt
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

func TestUpdate(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	var decoder = func(r *http.Request) *strava.UpdatableActivity {
		var act strava.UpdatableActivity
		dec := json.NewDecoder(r.Body)
		a.NoError(dec.Decode(&act))
		return &act
	}

	for _, tt := range []struct {
		id     int64
		name   string
		hidden bool
	}{
		{
			id:   1001,
			name: "empty update",
		},
		{
			id:     1002,
			name:   "hide",
			hidden: true,
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/1001", func(w http.ResponseWriter, r *http.Request) {
					a.Equal(http.MethodPut, r.Method)
					act := decoder(r)
					a.NotNil(act.Name)
					a.Nil(act.Description)
					a.False(*act.Hidden)
					http.ServeFile(w, r, "testdata/activity.json")
				})
				mux.HandleFunc("/activities/1002", func(w http.ResponseWriter, r *http.Request) {
					a.Equal(http.MethodPut, r.Method)
					act := decoder(r)
					a.True(*act.Hidden)
					http.ServeFile(w, r, "testdata/activity.json")
				})
			})
			defer svr.Close()
			update := &strava.UpdatableActivity{
				ID:     tt.id,
				Name:   &tt.name,
				Hidden: &tt.hidden,
			}
			act, err := client.Activity.Update(context.Background(), update)
			a.NoError(err)
			a.NotNil(act)
		})
	}
}
