package inreach_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/inreach"
)

func TestEncoding(t *testing.T) {
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
			a := assert.New(t)
			mux := http.NewServeMux()
			mux.HandleFunc("/Feed/Share/foobar", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "testdata/feed.kml")
			})
			svr := httptest.NewServer(mux)
			defer svr.Close()

			client, err := inreach.NewClient(inreach.WithBaseURL(svr.URL))
			a.NoError(err)
			feed, err := client.Feed.Feed(context.Background(), tt.user)
			a.NoError(err)
			a.NotNil(feed)

			x, err := feed.GeoJSON()
			a.NoError(err)
			a.NotNil(x)
		})
	}
}
