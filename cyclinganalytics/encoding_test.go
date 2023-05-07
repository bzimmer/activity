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
