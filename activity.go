package activity

import (
	"github.com/twpayne/go-gpx"
)

const UserAgent = "github.com/bzimmer/activity"

type GPXEncoder interface {
	// GPX returns a GPX instance
	GPX() (*gpx.GPX, error)
}
