package cyclinganalytics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/cyclinganalytics"
)

func TestRide(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		err    string
		rideID int64
	}{
		{
			rideID: 22322,
			name:   "query recent feed",
		},
		{
			rideID: 175334338355,
			name:   "query recent feed",
		},
		{
			rideID: 882722,
			name:   "query recent feed",
			err:    "Something went horribly wrong",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := http.NewServeMux()
			mux.HandleFunc("/ride/22322", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "testdata/ride.json")
			})
			mux.HandleFunc("/ride/175334338355", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "testdata/ride.json")
			})
			mux.HandleFunc("/ride/882722", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				http.ServeFile(w, r, "testdata/error.json")
			})

			svr := httptest.NewServer(mux)
			defer svr.Close()

			opts := cyclinganalytics.WithRideOptions(cyclinganalytics.RideOptions{
				Streams: []string{"latitude", "longitude", "elevation"},
			})
			client, err := cyclinganalytics.NewClient(
				cyclinganalytics.WithBaseURL(svr.URL),
				cyclinganalytics.WithHTTPTracing(false),
				cyclinganalytics.WithTokenCredentials("fooKey", "barToken", time.Time{}))
			a.NoError(err)
			ride, err := client.Rides.Ride(context.Background(), tt.rideID, opts)
			if tt.err != "" {
				a.Error(err)
				a.Nil(ride)
				a.Contains(err.Error(), tt.err)
				return
			}
			a.NoError(err)
			a.NotNil(ride)
			a.NotNil(ride.Streams)
			a.Equal(27154, len(ride.Streams.Elevation))
			gears := ride.Streams.Gears
			a.NotNil(gears)
			a.Equal(813, len(gears.Shifts))
		})
	}
}

func TestRides(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name string
		user cyclinganalytics.UserID
	}{
		{
			user: cyclinganalytics.Me,
			name: "query rides for `me`",
		},
		{
			user: cyclinganalytics.UserID(882782),
			name: "query rides for user",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := http.NewServeMux()
			mux.HandleFunc("/me/rides", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "testdata/me-rides.json")
			})
			mux.HandleFunc("/882782/rides", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "testdata/me-rides.json")
			})

			svr := httptest.NewServer(mux)
			defer svr.Close()

			client, err := cyclinganalytics.NewClient(
				cyclinganalytics.WithBaseURL(svr.URL),
				cyclinganalytics.WithHTTPTracing(false),
				cyclinganalytics.WithTokenCredentials("fooKey", "barToken", time.Time{}))
			a.NoError(err)
			rides, err := client.Rides.Rides(context.Background(), tt.user, activity.Pagination{})
			a.NoError(err)
			a.NotNil(rides)
			a.Len(rides, 2)
		})
	}
}
