package garmin_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity/garmin"
)

const tokenPath = "POST /personalapitoken-service/personal-api/personal-api-key/token"

// jwt returns an unsigned token whose exp claim is the given time.
func jwt(exp time.Time) string {
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc(fmt.Appendf(nil, `{"exp":%d}`, exp.Unix())) + ".sig"
}

// newClient serves the token endpoint, accepting the key "testAPIKey",
// along with whatever before adds.
func newClient(t *testing.T, before func(*http.ServeMux)) (*garmin.Client, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(tokenPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "ApiKey testAPIKey" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(jwt(time.Now().Add(time.Hour)) + "\n"))
	})
	if before != nil {
		before(mux)
	}
	svr := httptest.NewServer(mux)
	client, err := garmin.NewClient(
		garmin.WithAPIURL(svr.URL),
		garmin.WithHTTPTracing(false),
		garmin.WithAPIKey("testAPIKey"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return client, svr
}

func TestFault(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	f := &garmin.Fault{Message: "something went wrong", StatusCode: 400}
	a.Equal("something went wrong", f.Error())
	a.Equal(400, f.HTTPStatusCode())

	f2 := &garmin.Fault{}
	f2.SetDefaults(404, "Not Found")
	a.Equal(404, f2.StatusCode)
	a.Equal("Not Found", f2.Message)
}

func TestOptions(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name string
		opts []garmin.Option
		err  bool
	}{
		{name: "no options"},
		{name: "with api key", opts: []garmin.Option{garmin.WithAPIKey("key")}},
		{name: "with token", opts: []garmin.Option{garmin.WithToken(&oauth2.Token{AccessToken: "jwt"})}},
		{name: "with rate limiter", opts: []garmin.Option{garmin.WithRateLimiter(rate.NewLimiter(rate.Inf, 0))}},
		{name: "with nil rate limiter returns error", opts: []garmin.Option{garmin.WithRateLimiter(nil)}, err: true},
		{name: "with transport", opts: []garmin.Option{garmin.WithTransport(http.DefaultTransport)}},
		{name: "with nil transport returns error", opts: []garmin.Option{garmin.WithTransport(nil)}, err: true},
		{name: "with http client", opts: []garmin.Option{garmin.WithHTTPClient(http.DefaultClient)}},
		{name: "with nil http client returns error", opts: []garmin.Option{garmin.WithHTTPClient(nil)}, err: true},
		{name: "with http tracing", opts: []garmin.Option{garmin.WithHTTPTracing(true)}},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := garmin.NewClient(tt.opts...)
			if tt.err {
				a.Error(err)
				a.Nil(client)
				return
			}
			a.NoError(err)
			a.NotNil(client)
			a.NotNil(client.Exporter())
		})
	}
}

func TestNoCredentials(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, err := garmin.NewClient()
	a.NoError(err)
	_, err = client.Activities.File(t.Context(), 1)
	a.ErrorContains(err, "apiKey or accessToken required")
}

func TestInvalidAPIURL(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, err := garmin.NewClient(
		garmin.WithAPIURL("%%invalid"),
		garmin.WithToken(&oauth2.Token{AccessToken: "jwt"}),
	)
	a.NoError(err)
	_, err = client.Activities.File(t.Context(), 1)
	a.Error(err)
}
