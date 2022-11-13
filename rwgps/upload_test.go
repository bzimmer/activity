package rwgps_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/rwgps"
)

func TestUploadDone(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	u := &rwgps.Upload{TaskID: 10}
	a.Equal(activity.UploadID(10), u.Identifier())
	a.False(u.Done())
	a.True((&rwgps.Upload{Success: 1}).Done())
}
