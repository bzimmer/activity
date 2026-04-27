package strava_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity/strava"
)

func newClientMust(before func(*http.ServeMux), opts ...strava.Option) (*strava.Client, *httptest.Server) {
	client, svr, err := newClient(before, opts...)
	if err != nil {
		panic(err)
	}
	return client, svr
}

func newClient(before func(*http.ServeMux), opts ...strava.Option) (*strava.Client, *httptest.Server, error) {
	mux := http.NewServeMux()
	before(mux)
	svr := httptest.NewServer(mux)
	options := []strava.Option{
		strava.WithBaseURL(svr.URL),
		strava.WithHTTPTracing(false),
		strava.WithTokenCredentials("key", "token", time.Time{}),
	}
	if len(opts) > 0 {
		options = append(options, opts...)
	}
	client, err := strava.NewClient(options...)
	if err != nil {
		return nil, nil, err
	}
	return client, svr, nil
}

type ManyHandler struct {
	Filename string
	Total    int
	total    int
}

func (m *ManyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	n, err := strconv.Atoi(q.Get("per_page"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if m.Total > 0 {
		n = m.Total
		if m.total >= m.Total {
			n = 0
		}
	}
	m.total += n

	w.WriteHeader(http.StatusOK)
	data, err := os.ReadFile(m.Filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = w.Write([]byte("[")); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for i := 0; i < n; i++ {
		if _, err = w.Write(data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if i+1 < n {
			if _, err = w.Write([]byte(",")); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	if _, err = w.Write([]byte("]")); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func TestOptions(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tt := range []struct {
		name   string
		before func() []strava.Option
		after  func(client *strava.Client, err error)
	}{
		{
			name: "empty option",
			before: func() []strava.Option {
				return nil
			},
			after: func(client *strava.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with config",
			before: func() []strava.Option {
				return []strava.Option{strava.WithConfig(oauth2.Config{})}
			},
			after: func(client *strava.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with refresh",
			before: func() []strava.Option {
				return []strava.Option{strava.WithAutoRefresh(context.TODO())}
			},
			after: func(client *strava.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with rate limiter",
			before: func() []strava.Option {
				return []strava.Option{strava.WithRateLimiter(&rate.Limiter{})}
			},
			after: func(client *strava.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with token",
			before: func() []strava.Option {
				return []strava.Option{strava.WithToken(&oauth2.Token{})}
			},
			after: func(client *strava.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with transport",
			before: func() []strava.Option {
				return []strava.Option{strava.WithTransport(http.DefaultTransport)}
			},
			after: func(client *strava.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil transport returns error",
			before: func() []strava.Option {
				return []strava.Option{strava.WithTransport(nil)}
			},
			after: func(client *strava.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
		{
			name: "with http client",
			before: func() []strava.Option {
				return []strava.Option{strava.WithHTTPClient(http.DefaultClient)}
			},
			after: func(client *strava.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil http client returns error",
			before: func() []strava.Option {
				return []strava.Option{strava.WithHTTPClient(nil)}
			},
			after: func(client *strava.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr, err := newClient(func(_ *http.ServeMux) {}, tt.before()...)
			if err != nil {
				tt.after(client, err)
				return
			}
			defer svr.Close()
			tt.after(client, nil)
		})
	}
}
