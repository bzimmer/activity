package rwgps_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity/rwgps"
)

func newClient(before func(*http.ServeMux)) (*rwgps.Client, *httptest.Server) {
	mux := http.NewServeMux()
	if before != nil {
		before(mux)
	}
	svr := httptest.NewServer(mux)
	client, err := rwgps.NewClient(rwgps.WithBaseURL(svr.URL),
		rwgps.WithHTTPTracing(false),
		rwgps.WithClientCredentials("fooKey", ""),
		rwgps.WithTokenCredentials("barToken", "", time.Time{}),
	)
	if err != nil {
		panic(err)
	}
	return client, svr
}

func TestFault(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	f := func() error {
		return &rwgps.Fault{Message: "something went wrong"}
	}
	err := f()
	a.Error(err)
	a.Equal("something went wrong", err.Error())
}

func TestOptions(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func() []rwgps.Option
		after  func(client *rwgps.Client, err error)
	}{
		{
			name: "no options",
			before: func() []rwgps.Option {
				return nil
			},
			after: func(client *rwgps.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with config",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithConfig(oauth2.Config{})}
			},
			after: func(client *rwgps.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with token",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithToken(&oauth2.Token{})}
			},
			after: func(client *rwgps.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with rate limiter",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithRateLimiter(rate.NewLimiter(rate.Inf, 0))}
			},
			after: func(client *rwgps.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil rate limiter returns error",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithRateLimiter(nil)}
			},
			after: func(client *rwgps.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
		{
			name: "with transport",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithTransport(http.DefaultTransport)}
			},
			after: func(client *rwgps.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil transport returns error",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithTransport(nil)}
			},
			after: func(client *rwgps.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
		{
			name: "with http client",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithHTTPClient(http.DefaultClient)}
			},
			after: func(client *rwgps.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil http client returns error",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithHTTPClient(nil)}
			},
			after: func(client *rwgps.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
		{
			name: "with http tracing",
			before: func() []rwgps.Option {
				return []rwgps.Option{rwgps.WithHTTPTracing(true)}
			},
			after: func(client *rwgps.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := rwgps.NewClient(tt.before()...)
			tt.after(client, err)
		})
	}
}

func TestFaultFromServer(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(err error)
	}{
		{
			name: "fault with message from JSON body",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/users/current.json", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					enc := json.NewEncoder(w)
					_ = enc.Encode(&rwgps.Fault{Code: 401, Message: "unauthorized"})
				})
			},
			after: func(err error) {
				a.Error(err)
				a.Equal("unauthorized", err.Error())
			},
		},
		{
			name: "fault with defaults from empty body",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/users/current.json", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusForbidden)
				})
			},
			after: func(err error) {
				a.Error(err)
				a.Equal("Forbidden", err.Error())
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			user, err := client.Users.AuthenticatedUser(t.Context())
			a.Nil(user)
			tt.after(err)
		})
	}
}
