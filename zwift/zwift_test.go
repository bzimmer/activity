package zwift_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

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

func tokenMux(t *testing.T, tokens, profiles chan<- int) *http.ServeMux {
	a := assert.New(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			a.Error(r.Context().Err())
		case tokens <- 1:
		}
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
		select {
		case <-r.Context().Done():
			a.Error(r.Context().Err())
		case profiles <- 1:
		}
		enc := json.NewEncoder(w)
		a.NoError(enc.Encode(&zwift.Profile{FirstName: "barney"}))
	})
	return mux
}

func TestTokenRefresh(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name               string
		username, password string
		iterations         int
		token              bool
		err                string
	}{
		{
			name:       "success 100",
			iterations: 100,
			username:   "foo-user",
			password:   "bar-pass",
		},
		{
			name:       "no credentials",
			iterations: 10,
			err:        "accessToken required",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tokens := make(chan int)
			profiles := make(chan int)

			svr := httptest.NewServer(tokenMux(t, tokens, profiles))
			defer svr.Close()

			endpoint := zwift.Endpoint()
			endpoint.AuthURL = svr.URL + "/auth"
			endpoint.TokenURL = svr.URL + "/token"

			var opt zwift.Option
			if tt.token {
				opt = zwift.WithTokenCredentials("foo", "bar", time.Now().Add(time.Hour*24))
			} else {
				opt = zwift.WithTokenRefresh(tt.username, tt.password)
			}

			client, err := zwift.NewClient(
				opt,
				zwift.WithBaseURL(svr.URL),
				zwift.WithConfig(oauth2.Config{Endpoint: endpoint}),
			)
			a.NoError(err)
			a.NotNil(client)

			grp, ctx := errgroup.WithContext(context.Background())
			for i := 0; i < tt.iterations; i++ {
				grp.Go(func() error {
					profile, egg := client.Profile.Profile(ctx, "abcxyz")
					if egg != nil {
						return egg
					}
					if profile.FirstName != "barney" {
						return errors.New("wrong first name")
					}
					return nil
				})
			}
			err = grp.Wait()
			if tt.err != "" {
				a.Error(err)
				a.Contains(err.Error(), tt.err)
				return
			}
			a.NoError(err)
			close(tokens)
			close(profiles)
			v := 0
			for x := range tokens {
				v += x
			}
			if tt.token {
				a.Equal(0, v)
			} else {
				a.Equal(1, v)
			}
			v = 0
			for x := range profiles {
				v += x
			}
			a.Equal(tt.iterations, v)
		})
	}
}
