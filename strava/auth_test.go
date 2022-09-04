package strava_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestRefresh(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(token *oauth2.Token, err error)
	}{
		{
			name: "valid refresh",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/athletes/88273/stats", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/refresh.json")
				})
			},
			after: func(token *oauth2.Token, err error) {
				a.NoError(err)
				a.NotNil(token)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			token, err := client.Auth.Refresh(context.TODO())
			tt.after(token, err)
		})
	}
}
