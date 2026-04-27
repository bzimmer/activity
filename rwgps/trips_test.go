package rwgps_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/rwgps"
)

func TestTripUpload(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		file   *activity.File
		before func(mux *http.ServeMux)
		after  func(upload *rwgps.Upload, err error)
	}{
		{
			name:   "nil file returns error",
			file:   nil,
			before: func(_ *http.ServeMux) {},
			after: func(upload *rwgps.Upload, err error) {
				a.Error(err)
				a.Nil(upload)
			},
		},
		{
			name:   "file with no name returns error",
			file:   &activity.File{Format: activity.FormatGPX},
			before: func(_ *http.ServeMux) {},
			after: func(upload *rwgps.Upload, err error) {
				a.Error(err)
				a.Nil(upload)
			},
		},
		{
			name:   "file with original format returns error",
			file:   &activity.File{Name: "ride.gpx", Format: activity.FormatOriginal},
			before: func(_ *http.ServeMux) {},
			after: func(upload *rwgps.Upload, err error) {
				a.Error(err)
				a.Nil(upload)
			},
		},
		{
			name: "valid upload",
			file: &activity.File{
				Reader: strings.NewReader("<gpx/>"),
				Name:   "ride.gpx",
				Format: activity.FormatGPX,
			},
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/trips.json", func(w http.ResponseWriter, _ *http.Request) {
					enc := json.NewEncoder(w)
					_ = enc.Encode(&rwgps.Upload{TaskID: 9999, Success: 0})
				})
			},
			after: func(upload *rwgps.Upload, err error) {
				a.NoError(err)
				a.NotNil(upload)
				a.Equal(int64(9999), upload.TaskID)
			},
		},
		{
			name: "server error on upload",
			file: &activity.File{
				Reader: strings.NewReader("<gpx/>"),
				Name:   "ride.gpx",
				Format: activity.FormatGPX,
			},
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/trips.json", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			after: func(upload *rwgps.Upload, err error) {
				a.Error(err)
				a.Nil(upload)
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			upload, err := client.Trips.Upload(context.TODO(), tt.file)
			tt.after(upload, err)
		})
	}
}

func TestTrip(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(trip *rwgps.Trip, err error)
	}{
		{
			name: "valid trip",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/trips/94.json", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/rwgps_trip_94.json")
				})
			},
			after: func(trip *rwgps.Trip, err error) {
				a.NoError(err)
				a.NotNil(trip)
				a.Equal(rwgps.UserID(1), trip.UserID)
				a.Equal(rwgps.TypeTrip.String(), trip.Type)
				a.Equal(1465, len(trip.TrackPoints))
			},
		},
		{
			name: "invalid trip",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/trips/94.json", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			after: func(trip *rwgps.Trip, err error) {
				a.Error(err)
				a.Nil(trip)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			trip, err := client.Trips.Trip(context.TODO(), 94)
			tt.after(trip, err)
		})
	}
}

func TestRoute(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(route *rwgps.Trip, err error)
	}{
		{
			name: "valid trip",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/routes/94.json", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/rwgps_route_141014.json")
				})
			},
			after: func(route *rwgps.Trip, err error) {
				a.NoError(err)
				a.NotNil(route)
				a.Equal(1154, len(route.TrackPoints))
				a.Equal(int64(141014), route.ID)
				a.Equal(rwgps.TypeRoute.String(), route.Type)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			trip, err := client.Trips.Route(context.TODO(), 94)
			tt.after(trip, err)
		})
	}
}

func TestPagination(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		trips  bool
		routes bool
		before func(mux *http.ServeMux)
		after  func(trips []*rwgps.Trip, err error)
	}{
		{
			name:   "valid routes",
			routes: true,
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/users/88272/routes.json", func(w http.ResponseWriter, _ *http.Request) {
					enc := json.NewEncoder(w)
					a.NoError(enc.Encode(struct {
						Results []*rwgps.Trip `json:"results"`
					}{
						Results: []*rwgps.Trip{
							{ID: 10},
							{ID: 20},
						},
					}))
				})
			},
			after: func(trips []*rwgps.Trip, err error) {
				a.NoError(err)
				a.NotNil(trips)
				a.Len(trips, 2)
			},
		},
		{
			name:  "valid trips",
			trips: true,
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/users/88272/trips.json", func(w http.ResponseWriter, _ *http.Request) {
					enc := json.NewEncoder(w)
					a.NoError(enc.Encode(struct {
						Results []*rwgps.Trip `json:"results"`
					}{
						Results: []*rwgps.Trip{
							{ID: 110},
							{ID: 210},
						},
					}))
				})
			},
			after: func(trips []*rwgps.Trip, err error) {
				a.NoError(err)
				a.NotNil(trips)
				a.Len(trips, 2)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			var err error
			var trips []*rwgps.Trip
			if tt.trips {
				trips, err = client.Trips.Trips(context.TODO(), rwgps.UserID(88272), activity.Pagination{Total: 2})
			}
			if tt.routes {
				trips, err = client.Trips.Routes(context.TODO(), rwgps.UserID(88272), activity.Pagination{Total: 2})
			}
			tt.after(trips, err)
		})
	}
}

func TestStatus(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(status *rwgps.Upload, err error)
	}{
		{
			name: "valid status",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/queued_tasks/status.json", func(w http.ResponseWriter, r *http.Request) {
					dec := json.NewDecoder(r.Body)
					m := make(map[string]string)
					a.NoError(dec.Decode(&m))
					ids := m["ids"]
					a.NotEmpty(ids)
					n, err := strconv.ParseInt(ids, 10, 64)
					a.NoError(err)
					enc := json.NewEncoder(w)
					a.NoError(enc.Encode(&rwgps.Upload{
						TaskID:  n,
						Success: 1,
					}))
				})
			},
			after: func(status *rwgps.Upload, err error) {
				a.NoError(err)
				a.NotNil(status)
				a.Equal(int64(7818), status.TaskID)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			status, err := client.Trips.Status(context.TODO(), 7818)
			tt.after(status, err)
		})
	}
}
