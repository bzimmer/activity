package activity

import (
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-gpx"
)

const UserAgent = "github.com/bzimmer/activity"

type GPXEncoder interface {
	// GPX returns a GPX instance
	GPX() (*gpx.GPX, error)
}

type GeoJSONEncoder interface {
	// GeoJSON returns a GeoJSON instance
	GeoJSON() (*geojson.FeatureCollection, error)
}
