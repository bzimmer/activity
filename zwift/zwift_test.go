package zwift_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity/zwift"
)

func newClient(t *testing.T, mux *http.ServeMux, opts ...zwift.Option) (*zwift.Client, *httptest.Server) {
	a := assert.New(t)
	svr := httptest.NewServer(mux)

	endpoint := zwift.Endpoint()
	endpoint.AuthURL = svr.URL + "/auth"
	endpoint.TokenURL = svr.URL + "/token"

	client, err := zwift.NewClient(
		append(
			[]zwift.Option{
				zwift.WithBaseURL(svr.URL),
				zwift.WithConfig(oauth2.Config{Endpoint: endpoint}),
				zwift.WithTokenCredentials("foo", "bar", time.Now().Add(time.Hour*24)),
				zwift.WithClientCredentials("what", "now?"),
			},
			opts...,
		)...)
	a.NoError(err)
	a.NotNil(client)
	return client, svr
}

func TestFault(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name        string
		fault       *zwift.Fault
		wantMessage string
		wantCode    int
	}{
		{
			name:        "Error returns message",
			fault:       &zwift.Fault{Message: "something failed"},
			wantMessage: "something failed",
		},
		{
			name:        "SetDefaults fills empty fields",
			fault:       &zwift.Fault{},
			wantMessage: "Not Found",
			wantCode:    404,
		},
		{
			name:        "SetDefaults does not overwrite existing fields",
			fault:       &zwift.Fault{Code: 200, Message: "already set"},
			wantMessage: "already set",
			wantCode:    200,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fault.SetDefaults(404, "Not Found")
			a.Equal(tt.wantMessage, tt.fault.Error())
			if tt.wantCode != 0 {
				a.Equal(tt.wantCode, tt.fault.Code)
			}
		})
	}
}

func TestOptions(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func() []zwift.Option
		after  func(client *zwift.Client, err error)
	}{
		{
			name: "no options",
			before: func() []zwift.Option {
				return nil
			},
			after: func(client *zwift.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with token",
			before: func() []zwift.Option {
				return []zwift.Option{zwift.WithToken(&oauth2.Token{})}
			},
			after: func(client *zwift.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with auto refresh",
			before: func() []zwift.Option {
				return []zwift.Option{zwift.WithAutoRefresh(context.Background())}
			},
			after: func(client *zwift.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with rate limiter",
			before: func() []zwift.Option {
				return []zwift.Option{zwift.WithRateLimiter(rate.NewLimiter(rate.Inf, 0))}
			},
			after: func(client *zwift.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil rate limiter returns error",
			before: func() []zwift.Option {
				return []zwift.Option{zwift.WithRateLimiter(nil)}
			},
			after: func(client *zwift.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
		{
			name: "with transport",
			before: func() []zwift.Option {
				return []zwift.Option{zwift.WithTransport(http.DefaultTransport)}
			},
			after: func(client *zwift.Client, err error) {
				a.NoError(err)
				a.NotNil(client)
			},
		},
		{
			name: "with nil transport returns error",
			before: func() []zwift.Option {
				return []zwift.Option{zwift.WithTransport(nil)}
			},
			after: func(client *zwift.Client, err error) {
				a.Error(err)
				a.Nil(client)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := zwift.NewClient(tt.before()...)
			tt.after(client, err)
		})
	}
}

func TestExport(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	fitData := []byte("FIT_CONTENT")

	// Create an S3-like server to serve the FIT file
	s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Disposition", "filename=activity.fit")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fitData)
	}))
	defer s3Server.Close()

	s3Host := strings.TrimPrefix(s3Server.URL, "http://")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/profiles/me", func(w http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(w)
		a.NoError(enc.Encode(&zwift.Profile{ID: 1037, FirstName: "barney"}))
	})
	mux.HandleFunc("/api/profiles/1037/activities/882920", func(w http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(w)
		a.NoError(enc.Encode(&zwift.Activity{
			ID:            882920,
			FitFileBucket: "zwift-activity-prod",
			FitFileKey:    "test.fit",
		}))
	})

	// Custom transport that redirects S3 requests to s3Server
	transport := &redirectTransport{targetHost: s3Host}

	client, svr := newClient(t, mux, zwift.WithHTTPClient(&http.Client{Transport: transport}))
	defer svr.Close()

	export, err := client.Activity.Export(context.Background(), 882920)
	a.NoError(err)
	a.NotNil(export)
	a.Equal(int64(882920), export.ID)
	a.Equal("activity.fit", export.Name)
}

// redirectTransport redirects any *.s3.amazonaws.com request to the target host.
type redirectTransport struct {
	targetHost string
}

func (r *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasSuffix(req.URL.Host, ".s3.amazonaws.com") {
		newURL := *req.URL
		newURL.Scheme = "http"
		newURL.Host = r.targetHost
		newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL.String(), req.Body)
		if err != nil {
			return nil, err
		}
		newReq.Header = req.Header
		return http.DefaultTransport.RoundTrip(newReq)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func TestTokenRefresh(t *testing.T) {
	a := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
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
	mux.HandleFunc("/api/profiles/abcxyz", func(w http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(w)
		a.NoError(enc.Encode(&zwift.Profile{FirstName: "barney"}))
	})

	tests := []struct {
		name               string
		username, password string
		err                string
	}{
		{
			name:     "success",
			username: "foo-user",
			password: "bar-pass",
		},
		{
			name: "failure",
			err:  "accessToken required",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			svr := httptest.NewServer(mux)
			defer svr.Close()

			endpoint := zwift.Endpoint()
			endpoint.AuthURL = svr.URL + "/auth"
			endpoint.TokenURL = svr.URL + "/token"

			client, err := zwift.NewClient(
				zwift.WithTokenRefresh(tt.username, tt.password),
				zwift.WithBaseURL(svr.URL),
				zwift.WithConfig(oauth2.Config{Endpoint: endpoint}),
			)
			a.NoError(err)
			a.NotNil(client)

			ctx := context.Background()
			profile, err := client.Profile.Profile(ctx, "abcxyz")
			switch {
			case tt.err != "":
				a.Nil(profile)
				a.Error(err)
				a.Equal(tt.err, err.Error())
			default:
				a.NoError(err)
				a.NotNil(profile)
				a.Equal("barney", profile.FirstName)
			}
		})
	}
}
