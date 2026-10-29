package cyclinganalytics_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/cyclinganalytics"
)

func TestRideGPX(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	data, err := os.ReadFile("testdata/ride.json")
	a.NoError(err)
	var ride cyclinganalytics.Ride
	err = json.Unmarshal(data, &ride)
	a.NoError(err)
	a.NotNil(ride.Streams)
	a.Equal(5, len(ride.Streams.Elevation))
	a.Equal(5, len(ride.Streams.Latitude))
	a.Equal(5, len(ride.Streams.Longitude))

	gpx, err := ride.GPX()
	a.NoError(err)
	a.NotNil(gpx)
	a.Equal(5, len(gpx.Trk[0].TrkSeg[0].TrkPt))
	a.Equal(0, len(gpx.Rte))
}

func TestRideGPXMalformedStreams(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name    string
		streams cyclinganalytics.Streams
		points  int
		err     bool
	}{
		{
			name:    "latitude without longitude",
			streams: cyclinganalytics.Streams{Latitude: []float64{47, 47.1}},
			err:     true,
		},
		{
			name:    "longitude without latitude",
			streams: cyclinganalytics.Streams{Longitude: []float64{8, 8.1}},
			err:     true,
		},
		{
			name:    "longitude shorter than latitude",
			streams: cyclinganalytics.Streams{Latitude: []float64{47, 47.1, 47.2}, Longitude: []float64{8, 8.1}},
			points:  2,
		},
		{
			name: "elevation shorter than latitude",
			streams: cyclinganalytics.Streams{
				Latitude: []float64{47, 47.1}, Longitude: []float64{8, 8.1}, Elevation: []float64{100}},
			points: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ride := cyclinganalytics.Ride{Streams: tt.streams}
			x, err := ride.GPX()
			if tt.err {
				a.Error(err)
				a.Nil(x)
				return
			}
			a.NoError(err)
			a.Len(x.Trk[0].TrkSeg[0].TrkPt, tt.points)
		})
	}
}
