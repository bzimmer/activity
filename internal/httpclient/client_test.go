package httpclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity/internal/httpclient"
)

// testFaultError is a minimal Fault implementation used only in tests.
type testFaultError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (f *testFaultError) Error() string { return f.Message }

func (f *testFaultError) SetDefaults(code int, message string) {
	if f.Code == 0 {
		f.Code = code
	}
	if f.Message == "" {
		f.Message = message
	}
}

func newClient(before func(*http.ServeMux)) (*httpclient.Client[*testFaultError], *httptest.Server) {
	mux := http.NewServeMux()
	if before != nil {
		before(mux)
	}
	svr := httptest.NewServer(mux)
	c := httpclient.New(func() *testFaultError { return &testFaultError{} })
	return c, svr
}

func TestNew(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	c := httpclient.New(func() *testFaultError { return &testFaultError{} })
	a.NotNil(c)
	a.NotNil(c.HTTP)
	a.NotNil(c.Token)
}

func TestDo(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	type result struct {
		Foo string `json:"foo"`
	}

	tests := []struct {
		name   string
		before func(*http.ServeMux)
		after  func(*result, error)
	}{
		{
			name: "success",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) {
					enc := json.NewEncoder(w)
					a.NoError(enc.Encode(&result{Foo: "bar"}))
				})
			},
			after: func(res *result, err error) {
				a.NoError(err)
				a.NotNil(res)
				a.Equal("bar", res.Foo)
			},
		},
		{
			name: "success with nil v",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/nil", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				})
			},
			after: func(_ *result, err error) {
				a.NoError(err)
			},
		},
		{
			name: "error 4xx with JSON body",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/bad", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					enc := json.NewEncoder(w)
					a.NoError(enc.Encode(&testFaultError{Code: 400, Message: "bad request"}))
				})
			},
			after: func(_ *result, err error) {
				a.Error(err)
				a.Equal("bad request", err.Error())
			},
		},
		{
			name: "error 4xx with empty body",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/empty", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			after: func(_ *result, err error) {
				a.Error(err)
				a.Equal("Not Found", err.Error())
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, svr := newClient(tt.before)
			defer svr.Close()

			var path string
			switch tt.name {
			case "success":
				path = "/ok"
			case "success with nil v":
				path = "/nil"
			case "error 4xx with JSON body":
				path = "/bad"
			case "error 4xx with empty body":
				path = "/empty"
			}

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, svr.URL+path, nil)
			a.NoError(err)

			if tt.name == "success with nil v" {
				err = c.Do(req, nil)
				tt.after(nil, err)
			} else {
				var res result
				err = c.Do(req, &res)
				tt.after(&res, err)
			}
		})
	}
}

func TestDoDecodeErrors(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "invalid JSON in error body",
			status: http.StatusBadRequest,
			body:   "not-json",
		},
		{
			name:   "invalid JSON in success body",
			status: http.StatusOK,
			body:   "not-json",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, svr := newClient(func(mux *http.ServeMux) {
				mux.HandleFunc("/decode-err", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(tt.body))
				})
			})
			defer svr.Close()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, svr.URL+"/decode-err", nil)
			a.NoError(err)
			var res struct{ Foo string }
			err = c.Do(req, &res)
			a.Error(err)
		})
	}
}

func TestDoContextCancelled(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	c, svr := newClient(func(mux *http.ServeMux) {
		mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
			w.WriteHeader(http.StatusOK)
		})
	})
	defer svr.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svr.URL+"/slow", nil)
	a.NoError(err)

	var res struct{}
	err = c.Do(req, &res)
	a.Error(err)
	a.Equal(context.Canceled, err)
}

func TestApplyRateLimiter(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name    string
		limiter *rate.Limiter
		wantErr bool
	}{
		{
			name:    "valid limiter",
			limiter: rate.NewLimiter(rate.Inf, 0),
			wantErr: false,
		},
		{
			name:    "nil limiter",
			limiter: nil,
			wantErr: true,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := httpclient.New(func() *testFaultError { return &testFaultError{} })
			err := httpclient.ApplyRateLimiter(c, tt.limiter)
			if tt.wantErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
		})
	}
}

func TestApplyHTTPTracing(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name  string
		debug bool
	}{
		{name: "debug off", debug: false},
		{name: "debug on", debug: true},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := httpclient.New(func() *testFaultError { return &testFaultError{} })
			a.NoError(httpclient.ApplyHTTPTracing(c, tt.debug))
		})
	}
}

func TestApplyTransport(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name      string
		transport http.RoundTripper
		wantErr   bool
	}{
		{
			name:      "valid transport",
			transport: http.DefaultTransport,
			wantErr:   false,
		},
		{
			name:      "nil transport",
			transport: nil,
			wantErr:   true,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := httpclient.New(func() *testFaultError { return &testFaultError{} })
			err := httpclient.ApplyTransport(c, tt.transport)
			if tt.wantErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
		})
	}
}

func TestApplyHTTPClient(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name    string
		client  *http.Client
		wantErr bool
	}{
		{
			name:    "valid client",
			client:  http.DefaultClient,
			wantErr: false,
		},
		{
			name:    "nil client",
			client:  nil,
			wantErr: true,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := httpclient.New(func() *testFaultError { return &testFaultError{} })
			err := httpclient.ApplyHTTPClient(c, tt.client)
			if tt.wantErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
		})
	}
}

func TestApplyAutoRefresh(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	c := httpclient.New(func() *testFaultError { return &testFaultError{} })
	a.NoError(httpclient.ApplyAutoRefresh(context.Background(), c))
	a.NotNil(c.HTTP)
}
