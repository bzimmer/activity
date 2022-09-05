package rwgps_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/rwgps"
)

func TestTrip(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name    string
		context context.Context
		before  func(mux *http.ServeMux)
		after   func(trip *rwgps.Trip, err error)
	}{
		{
			name:    "valid trip",
			context: context.TODO(),
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
			name:    "invalid trip",
			context: context.TODO(),
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
		{
			name:    "nil context",
			context: nil,
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
			client, svr := newClientMux(tt.before)
			defer svr.Close()
			trip, err := client.Trips.Trip(tt.context, 94)
			tt.after(trip, err)
		})
	}
}

func TestRoute(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name    string
		context context.Context
		before  func(mux *http.ServeMux)
		after   func(route *rwgps.Trip, err error)
	}{
		{
			name:    "valid trip",
			context: context.TODO(),
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
			client, svr := newClientMux(tt.before)
			defer svr.Close()
			trip, err := client.Trips.Route(tt.context, 94)
			tt.after(trip, err)
		})
	}
}
