package inreach_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bzimmer/activity/inreach"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	a := assert.New(t)
	tests := []struct {
		name    string
		trace   bool
		client  *http.Client
		tripper http.RoundTripper
		err     string
	}{
		{
			name:    "all valid",
			trace:   true,
			client:  http.DefaultClient,
			tripper: http.DefaultTransport,
		},
		{
			name:    "nil client",
			client:  nil,
			tripper: http.DefaultTransport,
			err:     "nil client",
		},
		{
			name:    "nil transport",
			client:  http.DefaultClient,
			tripper: nil,
			err:     "nil transport",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			client, err := inreach.NewClient(
				inreach.WithHTTPTracing(false),
				inreach.WithTransport(tt.tripper),
				inreach.WithHTTPClient(tt.client))
			if tt.err != "" {
				a.Error(err)
				a.Contains(err.Error(), tt.err)
			} else {
				a.NoError(err)
				a.NotNil(client)
			}
		})
	}
}

func TestNotFound(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name string
		user string
	}{
		{
			user: "foobar",
			name: "query recent feed",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/Feed/Share/foobar", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "file-does-not-exist.kml")
			})

			svr := httptest.NewServer(mux)
			defer svr.Close()

			client, err := inreach.NewClient(inreach.WithBaseURL(svr.URL))
			a.NoError(err)
			feed, err := client.Feed.Feed(context.Background(), tt.user)
			a.Error(err)
			a.Nil(feed)
			a.Contains(err.Error(), http.StatusText(http.StatusNotFound))
		})
	}
}
