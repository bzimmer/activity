package hammerhead_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/hammerhead"
)

func TestActivities(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(acts []*hammerhead.ActivitySummary, err error)
	}{
		{
			name: "valid activities",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/hammerhead_activities.json")
				})
			},
			after: func(acts []*hammerhead.ActivitySummary, err error) {
				a.NoError(err)
				a.Len(acts, 2)
				a.Equal("activity-001", acts[0].ID)
				a.Equal("Morning Ride", acts[0].Name)
				a.Equal(3600, acts[0].Duration)
			},
		},
		{
			name: "server error",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			after: func(acts []*hammerhead.ActivitySummary, err error) {
				a.Error(err)
				a.Nil(acts)
			},
		},
		{
			name: "activities with start date",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities", func(w http.ResponseWriter, r *http.Request) {
					a.Equal("2024-01-01", r.URL.Query().Get("startDate"))
					enc := json.NewEncoder(w)
					a.NoError(enc.Encode(&hammerhead.ActivitiesPage{
						TotalItems:  1,
						TotalPages:  1,
						PerPage:     100,
						CurrentPage: 1,
						Data: []*hammerhead.ActivitySummary{
							{ID: "activity-003", Name: "New Year Ride", Duration: 1800},
						},
					}))
				})
			},
			after: func(acts []*hammerhead.ActivitySummary, err error) {
				a.NoError(err)
				a.Len(acts, 1)
				a.Equal("activity-003", acts[0].ID)
			},
		},
		{
			name: "pagination across multiple pages",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities", func(w http.ResponseWriter, r *http.Request) {
					page := r.URL.Query().Get("page")
					enc := json.NewEncoder(w)
					if page == "1" {
						a.NoError(enc.Encode(&hammerhead.ActivitiesPage{
							TotalItems:  2,
							TotalPages:  2,
							PerPage:     1,
							CurrentPage: 1,
							Data:        []*hammerhead.ActivitySummary{{ID: "a1", Name: "Ride 1"}},
						}))
					} else {
						a.NoError(enc.Encode(&hammerhead.ActivitiesPage{
							TotalItems:  2,
							TotalPages:  2,
							PerPage:     1,
							CurrentPage: 2,
							Data:        []*hammerhead.ActivitySummary{{ID: "a2", Name: "Ride 2"}},
						}))
					}
				})
			},
			after: func(acts []*hammerhead.ActivitySummary, err error) {
				a.NoError(err)
				a.Len(acts, 2)
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var startDate string
			if strings.Contains(tt.name, "start date") {
				startDate = "2024-01-01"
			}
			client, svr := newClient(t, tt.before)
			defer svr.Close()
			acts, err := client.Activities.Activities(t.Context(), activity.Pagination{}, startDate)
			tt.after(acts, err)
		})
	}
}

func TestActivity(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(act *hammerhead.Activity, err error)
	}{
		{
			name: "valid activity",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/activity-001", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/hammerhead_activity.json")
				})
			},
			after: func(act *hammerhead.Activity, err error) {
				a.NoError(err)
				a.NotNil(act)
				a.Equal("activity-001", act.ID)
				a.Equal("Morning Ride", act.Name)
				a.Equal(hammerhead.ActivityTypeRide, act.ActivityType)
				a.Equal("A lovely morning ride", act.Description)
			},
		},
		{
			name: "not found",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/missing", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			after: func(act *hammerhead.Activity, err error) {
				a.Error(err)
				a.Nil(act)
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, tt.before)
			defer svr.Close()
			var id string
			if strings.Contains(tt.name, "not found") {
				id = "missing"
			} else {
				id = "activity-001"
			}
			act, err := client.Activities.Activity(t.Context(), id)
			tt.after(act, err)
		})
	}
}

func TestFile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(file *activity.File, err error)
	}{
		{
			name: "valid fit file",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/activity-001/file", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/vnd.ant.fit")
					_, _ = w.Write([]byte("FIT file content"))
				})
			},
			after: func(file *activity.File, err error) {
				a.NoError(err)
				a.NotNil(file)
				a.Equal(activity.FormatFIT, file.Format)
				a.Equal("activity-001", file.Name)
				a.NoError(file.Close())
			},
		},
		{
			name: "server error on file",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/activity-001/file", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			after: func(file *activity.File, err error) {
				a.Error(err)
				a.Nil(file)
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, tt.before)
			defer svr.Close()
			file, err := client.Activities.File(t.Context(), "activity-001")
			tt.after(file, err)
		})
	}
}

func TestExporter(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(export *activity.Export, err error)
	}{
		{
			name: "valid export",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/12345/file", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/vnd.ant.fit")
					_, _ = w.Write([]byte("FIT file content"))
				})
			},
			after: func(export *activity.Export, err error) {
				a.NoError(err)
				a.NotNil(export)
				a.Equal(int64(12345), export.ID)
				a.Equal(activity.FormatFIT, export.Format)
			},
		},
		{
			name: "file error propagated",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities/12345/file", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			after: func(export *activity.Export, err error) {
				a.Error(err)
				a.Nil(export)
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, tt.before)
			defer svr.Close()
			exporter := client.Exporter()
			export, err := exporter.Export(t.Context(), 12345)
			tt.after(export, err)
		})
	}
}

func TestMissingAccessToken(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, err := hammerhead.NewClient(
		hammerhead.WithClientCredentials("id", "secret"),
	)
	a.NoError(err)

	_, err = client.Activities.Activities(t.Context(), activity.Pagination{}, "")
	a.Error(err)
	a.Contains(err.Error(), "accessToken required")

	_, err = client.Activities.Activity(t.Context(), "activity-001")
	a.Error(err)
	a.Contains(err.Error(), "accessToken required")

	_, err = client.Activities.File(t.Context(), "activity-001")
	a.Error(err)
	a.Contains(err.Error(), "accessToken required")
}

func TestActivitiesTruncation(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, svr := newClient(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/activities", func(w http.ResponseWriter, _ *http.Request) {
			enc := json.NewEncoder(w)
			a.NoError(enc.Encode(&hammerhead.ActivitiesPage{
				TotalItems:  2,
				TotalPages:  1,
				PerPage:     2,
				CurrentPage: 1,
				Data: []*hammerhead.ActivitySummary{
					{ID: "a1", Name: "Ride 1"},
					{ID: "a2", Name: "Ride 2"},
				},
			}))
		})
	})
	defer svr.Close()

	acts, err := client.Activities.Activities(t.Context(), activity.Pagination{Total: 1}, "")
	a.NoError(err)
	a.Len(acts, 1)
	a.Equal("a1", acts[0].ID)
}

func TestFileTransportError(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, svr := newClient(t, nil)
	svr.Close()

	_, err := client.Activities.File(t.Context(), "activity-001")
	a.Error(err)
}
