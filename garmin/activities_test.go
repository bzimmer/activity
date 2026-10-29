package garmin_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/garmin"
)

const (
	searchPath   = "GET /activitylist-service/activities/search/activities"
	downloadPath = "GET /download-service/files/activity/{id}"
)

func TestActivities(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		spec   activity.Pagination
		since  time.Time
		before func(mux *http.ServeMux)
		after  func(acts []*garmin.ActivitySummary, err error)
	}{
		{
			name: "valid activities",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc(searchPath, func(w http.ResponseWriter, r *http.Request) {
					a.Equal("Bearer", r.Header.Get("Authorization")[:6])
					http.ServeFile(w, r, "testdata/activities.json")
				})
			},
			after: func(acts []*garmin.ActivitySummary, err error) {
				a.NoError(err)
				a.Len(acts, 2)
				a.Equal(int64(24834688244), acts[0].ActivityID)
				a.Equal("Locarno Trail Running", acts[0].ActivityName)
				a.Equal("trail_running", acts[0].ActivityType.TypeKey)
				a.Equal(time.Date(2026, 10, 24, 8, 0, 12, 0, time.UTC), acts[0].StartTimeGMT.Time)
				a.Equal(time.Date(2026, 10, 24, 10, 0, 12, 0, time.UTC), acts[0].StartTimeLocal.Time)
				a.InDelta(24123.4, acts[0].Distance.Meters(), 0.01)
				a.InDelta(0, acts[1].ElevationGain.Meters(), 0.01)
			},
		},
		{
			name:  "start date",
			since: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			before: func(mux *http.ServeMux) {
				mux.HandleFunc(searchPath, func(w http.ResponseWriter, r *http.Request) {
					a.Equal("2024-01-01", r.URL.Query().Get("startDate"))
					_, _ = w.Write([]byte("[]"))
				})
			},
			after: func(acts []*garmin.ActivitySummary, err error) {
				a.NoError(err)
				a.Empty(acts)
			},
		},
		{
			name: "pages by offset until a short page",
			spec: activity.Pagination{Count: 2},
			before: func(mux *http.ServeMux) {
				mux.HandleFunc(searchPath, func(w http.ResponseWriter, r *http.Request) {
					start, _ := strconv.Atoi(r.URL.Query().Get("start"))
					a.Equal("2", r.URL.Query().Get("limit"))
					switch start {
					case 0:
						_, _ = w.Write([]byte(`[{"activityId":1},{"activityId":2}]`))
					case 2:
						_, _ = w.Write([]byte(`[{"activityId":3}]`))
					default:
						t.Errorf("unexpected start %d", start)
					}
				})
			},
			after: func(acts []*garmin.ActivitySummary, err error) {
				a.NoError(err)
				a.Len(acts, 3)
				a.Equal(int64(3), acts[2].ActivityID)
			},
		},
		{
			name: "total truncates",
			spec: activity.Pagination{Total: 1},
			before: func(mux *http.ServeMux) {
				mux.HandleFunc(searchPath, func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/activities.json")
				})
			},
			after: func(acts []*garmin.ActivitySummary, err error) {
				a.NoError(err)
				a.Len(acts, 1)
			},
		},
		{
			name: "bad timestamp",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc(searchPath, func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(`[{"activityId":1,"startTimeGMT":"yesterday"}]`))
				})
			},
			after: func(acts []*garmin.ActivitySummary, err error) {
				a.Error(err)
				a.Nil(acts)
			},
		},
		{
			name: "null and empty timestamps",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc(searchPath, func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte(`[{"activityId":1,"startTimeGMT":null,"startTimeLocal":""}]`))
				})
			},
			after: func(acts []*garmin.ActivitySummary, err error) {
				a.NoError(err)
				a.True(acts[0].StartTimeGMT.IsZero())
				a.True(acts[0].StartTimeLocal.IsZero())
			},
		},
		{
			name: "rate limited with a plain text body",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc(searchPath, func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Retry-After", "2")
					http.Error(w, "slow down", http.StatusTooManyRequests)
				})
			},
			after: func(acts []*garmin.ActivitySummary, err error) {
				a.Nil(acts)
				var f *garmin.Fault
				a.True(errors.As(err, &f))
				a.Equal(http.StatusTooManyRequests, f.HTTPStatusCode())
				a.Equal(2*time.Second, f.RetryAfter)
			},
		},
		{
			name: "fault message from a JSON body",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc(searchPath, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"message":"bad startDate"}`))
				})
			},
			after: func(_ []*garmin.ActivitySummary, err error) {
				a.EqualError(err, "bad startDate")
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, tt.before)
			defer svr.Close()
			tt.after(client.Activities.Activities(t.Context(), tt.spec, tt.since))
		})
	}
}

// zipped returns a ZIP holding the named files, each containing its name.
func zipped(t *testing.T, names ...string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(name))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestFile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name  string
		id    int64
		after func(f *activity.File, err error)
	}{
		{
			name: "fit upload",
			id:   1,
			after: func(f *activity.File, err error) {
				a.NoError(err)
				defer f.Close()
				a.Equal("1_ACTIVITY.fit", f.Filename)
				a.Equal("1", f.Name)
				a.Equal(activity.FormatFIT, f.Format)
				b, err := io.ReadAll(f)
				a.NoError(err)
				a.Equal("dir/1_ACTIVITY.fit", string(b))
			},
		},
		{
			name: "older tcx upload keeps its format",
			id:   4,
			after: func(f *activity.File, err error) {
				a.NoError(err)
				a.Equal("run.tcx", f.Filename)
				a.Equal(activity.FormatTCX, f.Format)
			},
		},
		{
			name: "more than one file",
			id:   5,
			after: func(f *activity.File, err error) {
				a.Nil(f)
				a.ErrorContains(err, "activity 5: expected one file in the original upload, found 2")
			},
		},
		{
			name: "not a zip",
			id:   6,
			after: func(f *activity.File, err error) {
				a.Nil(f)
				a.ErrorContains(err, "reading the original upload")
			},
		},
		{
			name: "manual entry has no file",
			id:   2,
			after: func(f *activity.File, err error) {
				a.Nil(f)
				var fault *garmin.Fault
				a.True(errors.As(err, &fault))
				a.Equal(http.StatusNotFound, fault.StatusCode)
			},
		},
		{
			name: "pre-FIT activity is refused",
			id:   3,
			after: func(f *activity.File, err error) {
				a.Nil(f)
				var fault *garmin.Fault
				a.True(errors.As(err, &fault))
				a.Equal(http.StatusForbidden, fault.StatusCode)
			},
		},
	}
	before := func(mux *http.ServeMux) {
		mux.HandleFunc(downloadPath, func(w http.ResponseWriter, r *http.Request) {
			switch r.PathValue("id") {
			case "1":
				// Garmin's archives nest nothing, but a path must still come out as its base name
				_, _ = w.Write(zipped(t, "dir/", "dir/1_ACTIVITY.fit"))
			case "4":
				_, _ = w.Write(zipped(t, "run.tcx"))
			case "5":
				_, _ = w.Write(zipped(t, "a.fit", "b.fit"))
			case "6":
				_, _ = w.Write([]byte("not a zip"))
			case "3":
				w.WriteHeader(http.StatusForbidden)
			default:
				http.NotFound(w, r)
			}
		})
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, before)
			defer svr.Close()
			tt.after(client.Activities.File(t.Context(), tt.id))
		})
	}
}

func TestExport(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, svr := newClient(t, func(mux *http.ServeMux) {
		mux.HandleFunc(downloadPath, func(w http.ResponseWriter, r *http.Request) {
			if r.PathValue("id") != "7" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(zipped(t, "7_ACTIVITY.fit"))
		})
	})
	defer svr.Close()

	exp, err := client.Exporter().Export(t.Context(), 7)
	a.NoError(err)
	defer exp.Close()
	a.Equal(int64(7), exp.ID)
	a.Equal(activity.FormatFIT, exp.Format)

	exp, err = client.Exporter().Export(t.Context(), 8)
	a.Error(err)
	a.Nil(exp)
}
