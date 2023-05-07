package zwift_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/bzimmer/activity/zwift"
)

func newClient(t *testing.T, mux *http.ServeMux) (*zwift.Client, *httptest.Server) {
	a := assert.New(t)
	svr := httptest.NewServer(mux)

	endpoint := zwift.Endpoint()
	endpoint.AuthURL = svr.URL + "/auth"
	endpoint.TokenURL = svr.URL + "/token"

	client, err := zwift.NewClient(
		zwift.WithBaseURL(svr.URL),
		zwift.WithConfig(oauth2.Config{Endpoint: endpoint}),
		zwift.WithTokenCredentials("foo", "bar", time.Now().Add(time.Hour*24)),
		zwift.WithClientCredentials("what", "now?"))
	a.NoError(err)
	a.NotNil(client)
	return client, svr
}

func TestTokenRefresh(t *testing.T) {
	a := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		n, err := w.Write([]byte(`{
				"access_token":"11223344556677889900",
				"token_type":"bearer",
				"expires_in":3600,
				"refresh_token":"SomeRefreshToken",
				"scope":"user"
			  }`))
		a.Greater(n, 0)
		a.NoError(err)
	})
	mux.HandleFunc("/api/profiles/abcxyz", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		a.NoError(enc.Encode(&zwift.Profile{FirstName: "barney"}))
	})

	tests := []struct {
		name               string
		username, password string
	}{
		{
			name:     "success",
			username: "foo-user",
			password: "bar-pass",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			svr := httptest.NewServer(mux)
			defer svr.Close()

			endpoint := zwift.Endpoint()
			endpoint.AuthURL = svr.URL + "/auth"
			endpoint.TokenURL = svr.URL + "/token"

			var opt zwift.Option
			switch {
			case tt.username != "" && tt.password != "":
				opt = zwift.WithTokenRefresh(tt.username, tt.password)
			default:
				opt = zwift.WithTokenCredentials("foo", "bar", time.Now().Add(time.Hour*24))
			}

			client, err := zwift.NewClient(
				opt,
				zwift.WithBaseURL(svr.URL),
				zwift.WithConfig(oauth2.Config{Endpoint: endpoint}),
			)
			a.NoError(err)
			a.NotNil(client)

			ctx := context.Background()
			profile, err := client.Profile.Profile(ctx, "abcxyz")
			a.NoError(err)
			a.NotNil(profile)
			a.Equal("barney", profile.FirstName)
		})
	}
}
