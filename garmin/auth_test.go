package garmin_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/garmin"
)

func TestToken(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, svr := newClient(t, nil)
	defer svr.Close()
	token, err := client.Auth.Token(t.Context())
	a.NoError(err)
	a.NotEmpty(token.AccessToken)
	a.Equal("Bearer", token.TokenType)
	a.WithinDuration(time.Now().Add(time.Hour), token.Expiry, time.Minute)
}

func TestTokenErrors(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name    string
		key     string
		handler http.HandlerFunc
		after   func(token *oauth2.Token, err error)
	}{
		{
			name: "bad key",
			key:  "wrong",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			after: func(token *oauth2.Token, err error) {
				a.Nil(token)
				var f *garmin.Fault
				a.True(errors.As(err, &f))
				a.Equal(http.StatusUnauthorized, f.StatusCode)
			},
		},
		{
			name: "empty token",
			key:  "testAPIKey",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("\n"))
			},
			after: func(token *oauth2.Token, err error) {
				a.Nil(token)
				a.ErrorContains(err, "empty token")
			},
		},
		{
			name: "opaque token never expires",
			key:  "testAPIKey",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("opaque"))
			},
			after: func(token *oauth2.Token, err error) {
				a.NoError(err)
				a.Equal("opaque", token.AccessToken)
				a.True(token.Expiry.IsZero())
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := http.NewServeMux()
			mux.HandleFunc(tokenPath, tt.handler)
			svr := httptest.NewServer(mux)
			defer svr.Close()
			client, err := garmin.NewClient(garmin.WithAPIURL(svr.URL), garmin.WithAPIKey(tt.key))
			a.NoError(err)
			tt.after(client.Auth.Token(t.Context()))
		})
	}
}

// TestTokenRenewed checks an expired token is exchanged again before the
// next request, and a valid one is reused.
func TestTokenRenewed(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	exchanges := 0
	mux := http.NewServeMux()
	mux.HandleFunc(tokenPath, func(w http.ResponseWriter, _ *http.Request) {
		exchanges++
		// the first token has already expired
		exp := time.Now().Add(-time.Minute)
		if exchanges > 1 {
			exp = time.Now().Add(time.Hour)
		}
		_, _ = w.Write([]byte(jwt(exp)))
	})
	mux.HandleFunc("GET /activitylist-service/activities/search/activities", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("[]"))
	})
	svr := httptest.NewServer(mux)
	defer svr.Close()
	client, err := garmin.NewClient(garmin.WithAPIURL(svr.URL), garmin.WithAPIKey("testAPIKey"))
	a.NoError(err)

	_, err = client.Auth.Token(t.Context())
	a.NoError(err)
	for range 2 {
		_, err = client.Activities.Activities(t.Context(), activity.Pagination{}, time.Time{})
		a.NoError(err)
	}
	a.Equal(2, exchanges)
}
