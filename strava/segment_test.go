package strava_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/strava"
)

func TestSegment(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(segment *strava.Segment, err error)
	}{
		{
			name: "valid segment",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/segments/229781", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/segment.json")
				})
			},
			after: func(segment *strava.Segment, err error) {
				a.NoError(err)
				a.NotNil(segment)
				a.Equal(229781, segment.ID)
				a.Equal("Hawk Hill", segment.Name)
			},
		},
		{
			name:   "invalid segment",
			before: func(_ *http.ServeMux) {},
			after: func(_ *strava.Segment, err error) {
				a.Error(err)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClientMust(tt.before)
			defer svr.Close()
			tt.after(client.Segment.Segment(context.TODO(), 229781))
		})
	}
}

func TestSegments(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name       string
		pagination activity.Pagination
		after      func(segments []*strava.Segment, err error)
	}{
		{
			name:       "test total, start, and count",
			pagination: activity.Pagination{Total: 127, Start: 0, Count: 1},
			after: func(segments []*strava.Segment, err error) {
				a.NoError(err)
				a.NotNil(segments)
				a.Equal(127, len(segments))
			},
		},
		{
			name:       "test total and start",
			pagination: activity.Pagination{Total: 234, Start: 0},
			after: func(segments []*strava.Segment, err error) {
				a.NoError(err)
				a.NotNil(segments)
				a.Equal(234, len(segments))
			},
		},
		{
			name:       "test total and start less than PageSize",
			pagination: activity.Pagination{Total: 27, Start: 0},
			after: func(segments []*strava.Segment, err error) {
				a.NoError(err)
				a.NotNil(segments)
				a.Equal(27, len(segments))
			},
		},
		{
			name:       "negative test",
			pagination: activity.Pagination{Total: -1},
			after: func(segments []*strava.Segment, err error) {
				a.Error(err)
				a.Nil(segments)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClientMust(func(mux *http.ServeMux) {
				mux.Handle("/segments/starred", &ManyHandler{
					Filename: "testdata/segment.json",
				})
			})
			defer svr.Close()
			tt.after(client.Segment.Segments(context.TODO(), tt.pagination))
		})
	}
}
