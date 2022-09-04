package strava_test

import (
	"context"
	"math"
	"net/http"
	"testing"

	"github.com/bzimmer/activity/strava"
	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/assert"
)

func TestAthlete(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(ath *strava.Athlete, err error)
	}{
		{
			name: "valid athlete",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/athlete", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/athlete.json")
				})
			},
			after: func(ath *strava.Athlete, err error) {
				a.NoError(err)
				a.NotNil(ath)
				a.Equal(1122, ath.ID)
				a.Equal(1, len(ath.Bikes))
				a.Equal(1, len(ath.Shoes))
			},
		},
		{
			name: "athlete not authorized",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/athlete", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					http.ServeFile(w, r, "testdata/athlete_unauthorized.json")
				})
			},
			after: func(ath *strava.Athlete, err error) {
				a.Error(err)
				a.Nil(ath)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			athlete, err := client.Athlete.Athlete(context.TODO())
			tt.after(athlete, err)
		})
	}
}

func TestAthleteStats(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(stats *strava.Stats, err error)
	}{
		{
			name: "athlete stats",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/athletes/88273/stats", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/athlete_stats.json")
				})
			},
			after: func(stats *strava.Stats, err error) {
				a.NoError(err)
				a.NotNil(stats)
				a.Equal(float64(14492298), math.Trunc(stats.AllRideTotals.Distance.Meters()))
				a.Equal(unit.Duration(12441), stats.AllSwimTotals.ElapsedTime)
				a.Equal(float64(1597), math.Trunc(stats.BiggestClimbElevationGain.Meters()))
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			stats, err := client.Athlete.Stats(context.TODO(), 88273)
			tt.after(stats, err)
		})
	}
}
