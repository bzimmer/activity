package strava_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/strava"
)

func TestRoute(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	client, err := newClient(http.StatusOK, "route.json")
	a.NoError(err)
	ctx := context.Background()
	rte, err := client.Route.Route(ctx, 26587226)
	a.NoError(err)
	a.NotNil(rte)
	a.Equal(int64(26587226), rte.ID)
}

func TestRoutes(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	ctx := context.Background()
	client, err := strava.NewClient(
		strava.WithTransport(&ManyTransport{
			Filename: "testdata/route.json",
		}),
		strava.WithTokenCredentials("fooKey", "barToken", time.Time{}))
	a.NoError(err)

	// test total, start, and count
	// success: the requested number of routes because count/pagesize == 1
	acts, err := client.Route.Routes(ctx, 26587226, activity.Pagination{Total: 127, Start: 0, Count: 1})
	a.NoError(err)
	a.NotNil(acts)
	a.Equal(127, len(acts))

	// test total and start
	// success: the requested number of routes is exceeded because count/pagesize not specified
	acts, err = client.Route.Routes(ctx, 26587226, activity.Pagination{Total: 234, Start: 0})
	a.NoError(err)
	a.NotNil(acts)
	a.Equal(234, len(acts))

	// test total and start less than PageSize
	// success: the requested number of routes because count/pagesize <= strava.PageSize
	a.True(27 < strava.PageSize)
	acts, err = client.Route.Routes(ctx, 26587226, activity.Pagination{Total: 27, Start: 0})
	a.NoError(err)
	a.NotNil(acts)
	a.Equal(27, len(acts))

	// negative test
	acts, err = client.Route.Routes(ctx, 26587226, activity.Pagination{Total: -1})
	a.Error(err)
	a.Nil(acts)
}
