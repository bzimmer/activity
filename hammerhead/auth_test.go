package hammerhead_test

import (
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
				mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
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
			client, svr := newClient(t, tt.before)
			defer svr.Close()
			token, err := client.Auth.Refresh(t.Context())
			tt.after(token, err)
		})
	}
}
