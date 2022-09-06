package rwgps_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/rwgps"
)

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
				mux.HandleFunc("/trips/94.json", func(w http.ResponseWriter, r *http.Request) {
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
				mux.HandleFunc("/users/88272/routes.json", func(w http.ResponseWriter, r *http.Request) {
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
				mux.HandleFunc("/users/88272/trips.json", func(w http.ResponseWriter, r *http.Request) {
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
