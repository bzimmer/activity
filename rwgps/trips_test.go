package rwgps_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/rwgps"
)

func contextNil() context.Context {
	return nil
}

func TestTrip(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	c, err := newClient(http.StatusOK, "rwgps_trip_94.json")
	a.NoError(err)
	a.NotNil(c)

	ctx := context.Background()
	trk, err := c.Trips.Trip(ctx, 94)
	a.NoError(err)
	a.NotNil(trk)
	a.Equal(int64(94), trk.ID)
	a.Equal(rwgps.TypeTrip.String(), trk.Type)
	a.Equal(1465, len(trk.TrackPoints))

	trk, err = c.Trips.Trip(contextNil(), 94)
	a.Error(err)
	a.Nil(trk)

	c, err = newClient(http.StatusUnauthorized, "rwgps_trip_94.json")
	a.NoError(err)
	a.NotNil(c)
	trk, err = c.Trips.Trip(ctx, 94)
	a.Error(err)
	a.Nil(trk)
}

func TestRoute(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	c, err := newClient(http.StatusOK, "rwgps_route_141014.json")
	a.NoError(err)
	a.NotNil(c)

	ctx := context.Background()
	rte, err := c.Trips.Route(ctx, 141014)
	a.NoError(err)
	a.NotNil(rte)
	a.Equal(1154, len(rte.TrackPoints))
	a.Equal(int64(141014), rte.ID)
	a.Equal(rwgps.TypeRoute.String(), rte.Type)

	gpx, err := rte.GPX()
	a.NoError(err)
	a.NotNil(gpx)

	rte, err = c.Trips.Route(contextNil(), 141014)
	a.Error(err)
	a.Nil(rte)
}
