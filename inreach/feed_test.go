package inreach_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/inreach"
)

func TestFeed(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		name   string
		user   string
		before time.Time
		after  time.Time
	}{
		{
			user: "foobar",
			name: "query recent feed",
		},
		{
			user:   "datetimer",
			name:   "query recent feed",
			before: time.Now(),
			after:  time.Now().Add(time.Hour * -24),
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/Feed/Share/foobar", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "testdata/feed.kml")
			})
			mux.HandleFunc("/Feed/Share/datetimer", func(w http.ResponseWriter, r *http.Request) {
				d1, ok := r.URL.Query()["d1"]
				a.True(ok)
				a.NotNil(d1)
				a.Len(d1, 1)
				d1time, err := time.Parse(inreach.DateFormat, d1[0])
				a.NoError(err)
				d2, ok := r.URL.Query()["d2"]
				a.True(ok)
				a.NotNil(d2)
				a.Len(d2, 1)
				d2time, err := time.Parse(inreach.DateFormat, d2[0])
				a.NoError(err)
				a.True(d1time.Before(d2time))
				http.ServeFile(w, r, "testdata/feed.kml")
			})

			svr := httptest.NewServer(mux)
			defer svr.Close()

			opt := inreach.WithDateRange(tt.before, tt.after)

			client, err := inreach.NewClient(inreach.WithBaseURL(svr.URL))
			a.NoError(err)
			feed, err := client.Feed.Feed(context.Background(), tt.user, opt)
			a.NoError(err)
			a.NotNil(feed)
		})
	}
}
