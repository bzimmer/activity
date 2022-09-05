package rwgps_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/bzimmer/activity/rwgps"
)

func newClient(before func(*http.ServeMux)) (*rwgps.Client, *httptest.Server) {
	mux := http.NewServeMux()
	before(mux)
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
