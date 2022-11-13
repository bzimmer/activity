package rwgps_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/rwgps"
)

func TestTripEncoding(t *testing.T) {
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
				a.Equal(int64(94), trip.ID)
				a.Equal(rwgps.TypeTrip.String(), trip.Type)
				a.Equal(1465, len(trip.TrackPoints))

				gpx, err := trip.GPX()
				a.NoError(err)
				a.NotNil(gpx)
				a.Equal(1465, len(gpx.Trk[0].TrkSeg[0].TrkPt))
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
