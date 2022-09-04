package strava_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/strava"
)

func TestRoute(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(route *strava.Route, err error)
	}{
		{
			name: "valid route",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/routes/26587226", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/route.json")
				})
			},
			after: func(route *strava.Route, err error) {
				a.NoError(err)
				a.NotNil(route)
				a.Equal(int64(26587226), route.ID)
			},
		},
		{
			name:   "invalid route",
			before: func(mux *http.ServeMux) {},
			after: func(route *strava.Route, err error) {
				a.Error(err)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(tt.before)
			defer svr.Close()
			tt.after(client.Route.Route(context.TODO(), 26587226))
		})
	}
}

func TestRoutes(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name       string
		pagination activity.Pagination
		after      func(routes []*strava.Route, err error)
	}{
		{
			// success: the requested number of routes because count/pagesize == 1
			name:       "test total, start, and count",
			pagination: activity.Pagination{Total: 127, Start: 0, Count: 1},
			after: func(routes []*strava.Route, err error) {
				a.NoError(err)
				a.NotNil(routes)
				a.Equal(127, len(routes))
			},
		},
		{
			// success: the requested number of routes is exceeded because count/pagesize not specified
			name:       "test total and start",
			pagination: activity.Pagination{Total: 234, Start: 0},
			after: func(routes []*strava.Route, err error) {
				a.NoError(err)
				a.NotNil(routes)
				a.Equal(234, len(routes))
			},
		},
		{
			// success: the requested number of routes because count/pagesize <= strava.PageSize
			name:       "test total and start less than PageSize",
			pagination: activity.Pagination{Total: 27, Start: 0},
			after: func(routes []*strava.Route, err error) {
				a.NoError(err)
				a.NotNil(routes)
				a.Equal(27, len(routes))
			},
		},
		{
			name:       "negative test",
			pagination: activity.Pagination{Total: -1},
			after: func(routes []*strava.Route, err error) {
				a.Error(err)
				a.Nil(routes)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(func(mux *http.ServeMux) {
				mux.Handle("/athletes/26587226/routes", &ManyHandler{
					Filename: "testdata/route.json",
				})
			})
			defer svr.Close()
			tt.after(client.Route.Routes(context.TODO(), 26587226, tt.pagination))
		})
	}
}
