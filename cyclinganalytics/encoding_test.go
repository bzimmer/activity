package cyclinganalytics_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/cyclinganalytics"
)

func TestRideGPX(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	client, err := newClient(http.StatusOK, "ride.json")
	a.NoError(err)
	ctx := context.Background()
	opts := cyclinganalytics.RideOptions{
		Streams: []string{"latitude", "longitude", "elevation"},
	}
	ride, err := client.Rides.Ride(ctx, 175334338355, cyclinganalytics.WithRideOptions(opts))
	a.NoError(err)
	a.NotNil(ride)
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
