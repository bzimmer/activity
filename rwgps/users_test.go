package rwgps_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/rwgps"
)

func TestAthlete(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(ath *rwgps.User, err error)
	}{
		{
			name: "valid athlete",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/users/current.json", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/rwgps_users_1122.json")
				})
			},
			after: func(ath *rwgps.User, err error) {
				a.NoError(err)
				a.NotNil(ath)
				a.Equal(rwgps.UserID(1122), ath.ID)
			},
		},
		{
			name: "athlete not authorized",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/users/current.json", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
				})
			},
			after: func(ath *rwgps.User, err error) {
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
			athlete, err := client.Users.AuthenticatedUser(context.TODO())
			tt.after(athlete, err)
		})
	}
}
