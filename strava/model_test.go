package strava_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/strava"
)

func TestFault(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	f := func() error {
		return &strava.Fault{Message: "foo"}
	}
	err := f()
	a.Error(err)
	a.Equal("foo", err.Error())
}

// TestFaultHTTPStatusCode proves the real decode path -- not just that the type
// satisfies the interface -- by round-tripping an actual 429 through the client,
// using the body shape Strava sends for a rate limit: a message with no code.
func TestFaultHTTPStatusCode(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, svr := newClientMust(func(mux *http.ServeMux) {
		mux.HandleFunc("/athlete", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message": "Rate Limit Exceeded"}`))
		})
	})
	defer svr.Close()

	_, err := client.Athlete.Athlete(context.Background())
	a.Error(err)

	var fault *strava.Fault
	a.ErrorAs(err, &fault)
	a.Equal("Rate Limit Exceeded", fault.Message)
	a.Equal(http.StatusTooManyRequests, fault.HTTPStatusCode())

	var f interface{ HTTPStatusCode() int }
	a.ErrorAs(err, &f)
	a.Equal(http.StatusTooManyRequests, f.HTTPStatusCode())
}
