package strava_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bzimmer/activity/strava"
	"github.com/stretchr/testify/assert"
)

func TestGPXRoute(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	newMux := func() *http.ServeMux {
		mux := http.NewServeMux()
		mux.HandleFunc("/routes/26587226", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/route.json")
		})
		return mux
	}

	tests := []struct {
		id          int64
		name        string
		err         bool
		streams     []string
		tracks      int
		trackpoints int
		routes      int
		routepoints int
		desc        string
		link        string
	}{
		{
			id:          26587226,
			name:        "route with polyline",
			tracks:      0,
			routes:      1,
			trackpoints: 0,
			routepoints: 2076,
			desc:        "between Deer Park and Obstruction Point Road",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svr := httptest.NewServer(newMux())
			defer svr.Close()

			client, err := newTestClient(strava.WithBaseURL(svr.URL))
			a.NoError(err)
			rte, err := client.Route.Route(context.Background(), tt.id)
			a.NoError(err)
			a.NotNil(rte)

			gpx, err := rte.GPX()
			switch tt.err {
			case true:
				a.Error(err)
				a.Nil(gpx)
			case false:
				a.NoError(err)
				a.NotNil(gpx)
				a.Equal(tt.tracks, len(gpx.Trk))
				a.Equal(tt.routes, len(gpx.Rte))
				a.Equal(tt.routepoints, len(gpx.Rte[0].RtePt))
				if tt.desc != "" {
					a.Contains(gpx.Rte[0].Desc, tt.desc)
				}
			}
		})
	}
}

func TestGPXActivity(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	newMux := func() *http.ServeMux {
		mux := http.NewServeMux()
		mux.HandleFunc("/activities/12345", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/activity.json")
		})
		mux.HandleFunc("/activities/66282823", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/activity_with_polyline.json")
		})
		mux.HandleFunc("/activities/66282823/streams/latlng,altitude,time", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/streams.json")
		})
		return mux
	}

	tests := []struct {
		id          int64
		name        string
		err         bool
		streams     []string
		tracks      int
		trackpoints int
		routes      int
		routepoints int
		desc        string
	}{
		{
			id:   12345,
			name: "activity with no streams or polyline",
			err:  true,
		},
		{
			id:          66282823,
			name:        "activity with polyline",
			tracks:      1,
			trackpoints: 7,
		},
		{
			id:          66282823,
			name:        "activity with streams",
			tracks:      1,
			trackpoints: 1405,
			desc:        "Walk in the woods",
			streams:     []string{"latlng", "altitude", "time"},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svr := httptest.NewServer(newMux())
			defer svr.Close()

			client, err := newTestClient(strava.WithBaseURL(svr.URL))
			a.NoError(err)
			act, err := client.Activity.Activity(context.Background(), tt.id, tt.streams...)
			a.NoError(err)
			a.NotNil(act)

			gpx, err := act.GPX()
			switch tt.err {
			case true:
				a.Error(err)
				a.Nil(gpx)
			case false:
				a.NoError(err)
				a.NotNil(gpx)
				a.Equal(tt.tracks, len(gpx.Trk))
				a.Equal(tt.trackpoints, len(gpx.Trk[0].TrkSeg[0].TrkPt))
				a.Equal(tt.routes, len(gpx.Rte))
				if tt.desc != "" {
					a.Contains(gpx.Trk[0].Desc, tt.desc)
				}
				if tt.trackpoints > 0 {
					a.Equal(fmt.Sprintf("https://strava.com/activities/%d", tt.id), gpx.Trk[0].Link[0].HREF)
				}
			}
		})
	}
}
