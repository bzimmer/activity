package rwgps_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/rwgps"
)

func TestUser(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	c, err := newClient(http.StatusOK, "rwgps_users_1122.json")
	a.NoError(err)
	a.NotNil(c)

	ctx := context.Background()
	user, err := c.Users.AuthenticatedUser(ctx)
	a.NoError(err)
	a.NotNil(user)
	a.Equal(rwgps.UserID(1122), user.ID)
}
