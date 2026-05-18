package strava_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/strava"
)

func TestSegmentEffort(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		opts   []strava.Option
		after  func(segmentEffort *strava.SegmentEffort, err error)
	}{
		{
			name: "valid segment effort",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/segment_efforts/229781", func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, "testdata/segment_effort.json")
				})
			},
			after: func(segmentEffort *strava.SegmentEffort, err error) {
				a.NoError(err)
				a.NotNil(segmentEffort)
				a.Equal(int64(229781), segmentEffort.ID)
				a.Equal("Hawk Hill Effort", segmentEffort.Name)
				a.NotNil(segmentEffort.Segment)
				a.Equal(229781, segmentEffort.Segment.ID)
			},
		},
		{
			name:   "invalid segment effort",
			before: func(_ *http.ServeMux) {},
			after: func(_ *strava.SegmentEffort, err error) {
				a.Error(err)
			},
		},
		{
			name:   "invalid base url",
			before: func(_ *http.ServeMux) {},
			opts:   []strava.Option{strava.WithBaseURL("://bad-url")},
			after: func(_ *strava.SegmentEffort, err error) {
				a.Error(err)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClientMust(tt.before, tt.opts...)
			defer svr.Close()
			tt.after(client.Segment.SegmentEffort(context.TODO(), 229781))
		})
	}
}

func TestSegmentEfforts(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name       string
		pagination activity.Pagination
		opts       []strava.Option
		after      func(segmentEfforts []*strava.SegmentEffort, err error)
	}{
		{
			name:       "test total, start, and count",
			pagination: activity.Pagination{Total: 127, Start: 0, Count: 1},
			after: func(segmentEfforts []*strava.SegmentEffort, err error) {
				a.NoError(err)
				a.NotNil(segmentEfforts)
				a.Equal(127, len(segmentEfforts))
			},
		},
		{
			name:       "test total and start",
			pagination: activity.Pagination{Total: 234, Start: 0},
			after: func(segmentEfforts []*strava.SegmentEffort, err error) {
				a.NoError(err)
				a.NotNil(segmentEfforts)
				a.Equal(234, len(segmentEfforts))
			},
		},
		{
			name:       "test total and start less than PageSize",
			pagination: activity.Pagination{Total: 27, Start: 0},
			after: func(segmentEfforts []*strava.SegmentEffort, err error) {
				a.NoError(err)
				a.NotNil(segmentEfforts)
				a.Equal(27, len(segmentEfforts))
			},
		},
		{
			name:       "negative test",
			pagination: activity.Pagination{Total: -1},
			after: func(segmentEfforts []*strava.SegmentEffort, err error) {
				a.Error(err)
				a.Nil(segmentEfforts)
			},
		},
		{
			name:       "invalid base url",
			pagination: activity.Pagination{Total: 1},
			opts:       []strava.Option{strava.WithBaseURL("://bad-url")},
			after: func(segmentEfforts []*strava.SegmentEffort, err error) {
				a.Error(err)
				a.Nil(segmentEfforts)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClientMust(func(mux *http.ServeMux) {
				mux.Handle("/segment_efforts", &ManyHandler{
					Filename: "testdata/segment_effort.json",
				})
			}, tt.opts...)
			defer svr.Close()
			tt.after(client.Segment.SegmentEfforts(context.TODO(), tt.pagination))
		})
	}
}
