package strava_test

import (
	"testing"

	"github.com/bzimmer/activity/strava"
	"github.com/stretchr/testify/assert"
)

func TestFault(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	f := func() error {
		return &strava.Fault{Message: "foo"}
	}
	err := f()
	a.Error(err)
	a.Equal("foo", err.Error())
}
