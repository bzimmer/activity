package rwgps_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/rwgps"
)

func TestEncodingFromTrip(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	c, err := newClient(http.StatusOK, "rwgps_trip_94.json")
	a.NoError(err)
	a.NotNil(c)

	ctx := context.Background()
	trip, err := c.Trips.Trip(ctx, 94)
	a.NoError(err)
	a.NotNil(trip)
	a.Equal(int64(94), trip.ID)
	a.Equal(rwgps.TypeTrip.String(), trip.Type)
	a.Equal(1465, len(trip.TrackPoints))

	gpx, err := trip.GPX()
	a.NoError(err)
	a.NotNil(gpx)
	a.Equal(1465, len(gpx.Trk[0].TrkSeg[0].TrkPt))

	fc, err := trip.GeoJSON()
	a.NoError(err)
	a.NotNil(fc)
	a.Len(fc.Features, 1)
	a.Len(fc.Features[0].Geometry.FlatCoords(), 1465*3)
}
