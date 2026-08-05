package hammerhead_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/hammerhead"
)

func newClient(t *testing.T, before func(*http.ServeMux)) (*hammerhead.Client, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	if before != nil {
		before(mux)
	}
	svr := httptest.NewServer(mux)
	client, err := hammerhead.NewClient(
		hammerhead.WithAPIURL(svr.URL),
		hammerhead.WithAuthURL(svr.URL),
		hammerhead.WithHTTPTracing(false),
		hammerhead.WithClientCredentials("testClientID", "testClientSecret"),
		hammerhead.WithTokenCredentials("testAccessToken", "testRefreshToken", time.Time{}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return client, svr
}

func TestFault(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	f := &hammerhead.Fault{Message: "something went wrong", StatusCode: 400}
	a.Equal("something went wrong", f.Error())

	f2 := &hammerhead.Fault{}
	f2.SetDefaults(404, "Not Found")
	a.Equal(404, f2.StatusCode)
	a.Equal("Not Found", f2.Message)
}

func TestOptions(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func() []hammerhead.Option
		after  func(client *hammerhead.Client, err error)
	}{
		{
			name: "no options",
			before: func() []hammerhead.Option {
				return nil
			},
			after: func(client *hammerhead.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with config",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithConfig(oauth2.Config{})}
			},
			after: func(client *hammerhead.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with token",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithToken(&oauth2.Token{})}
			},
			after: func(client *hammerhead.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with auto refresh",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithAutoRefresh(context.Background())}
			},
			after: func(client *hammerhead.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with rate limiter",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithRateLimiter(rate.NewLimiter(rate.Inf, 0))}
			},
			after: func(client *hammerhead.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil rate limiter returns error",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithRateLimiter(nil)}
			},
			after: func(client *hammerhead.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
		{
			name: "with transport",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithTransport(http.DefaultTransport)}
			},
			after: func(client *hammerhead.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil transport returns error",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithTransport(nil)}
			},
			after: func(client *hammerhead.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
		{
			name: "with http client",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithHTTPClient(http.DefaultClient)}
			},
			after: func(client *hammerhead.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil http client returns error",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithHTTPClient(nil)}
			},
			after: func(client *hammerhead.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
		{
			name: "with http tracing",
			before: func() []hammerhead.Option {
				return []hammerhead.Option{hammerhead.WithHTTPTracing(true)}
			},
			after: func(client *hammerhead.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := hammerhead.NewClient(tt.before()...)
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
			name: "fault with defaults from empty body",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/activities", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
				})
			},
			after: func(err error) {
				a.Error(err)
				a.Equal("Unauthorized", err.Error())
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, tt.before)
			defer svr.Close()
			_, err := client.Activities.Activities(t.Context(), activity.Pagination{}, "")
			tt.after(err)
		})
	}
}

func TestEndpoint(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	ep := hammerhead.Endpoint()
	a.NotEmpty(ep.AuthURL)
	a.NotEmpty(ep.TokenURL)
}

func TestInvalidAPIURL(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, err := hammerhead.NewClient(
		hammerhead.WithAPIURL("%%invalid"),
		hammerhead.WithTokenCredentials("token", "refresh", time.Time{}),
	)
	a.NoError(err)
	_, err = client.Activities.Activity(t.Context(), "123")
	a.Error(err)
}
