package inreach

import (
	"strconv"
	"time"

	"github.com/bzimmer/activity"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

var _ activity.GeoJSONEncoder = (*Feed)(nil)

func (k *Feed) GeoJSON() (*geojson.FeatureCollection, error) { //nolint
	if k.Document == nil || k.Document.Folder == nil || k.Document.Folder.Placemark == nil {
		return &geojson.FeatureCollection{}, nil
	}
	var features []*geojson.Feature
	for _, pm := range k.Document.Folder.Placemark {
		// the KML has a Placemark for an unnecessary LineString
		if pm.ExtendedData == nil || len(pm.ExtendedData.Data) == 0 {
			continue
		}
		var id string
		c := make([]float64, 3)
		p := make(map[string]interface{})
		for _, ed := range pm.ExtendedData.Data {
			switch ed.Name {
			case FieldID:
				id = ed.Value
			case FieldLatitude:
				x, err := strconv.ParseFloat(ed.Value, 64)
				if err != nil {
					return nil, err
				}
				c[1] = x
			case FieldLongitude:
				x, err := strconv.ParseFloat(ed.Value, 64)
				if err != nil {
					return nil, err
				}
				c[0] = x
			case FieldElevation:
				if m := reElevation.FindStringSubmatch(ed.Value); m != nil {
					x, err := strconv.ParseFloat(m[1], 64)
					if err != nil {
						return nil, err
					}
					c[2] = x
				}
			case FieldIMEI:
				x, err := strconv.ParseInt(ed.Value, 0, 64)
				if err != nil {
					return nil, err
				}
				p[ed.Name] = x
			case FieldInEmergency, FieldValidGPS:
				x, err := strconv.ParseBool(ed.Value)
				if err != nil {
					return nil, err
				}
				p[ed.Name] = x
			case FieldTime:
				// use the UTC value and convert to local time when displayed
			case FieldTimeUTC:
				t, err := time.Parse(timeLayout, ed.Value)
				if err != nil {
					return nil, err
				}
				p[ed.Name] = t
			default:
				p[ed.Name] = ed.Value
			}
		}

		features = append(features, &geojson.Feature{
			ID:         id,
			Geometry:   geom.NewPointFlat(geom.XYZ, c),
			Properties: p,
		})
	}

	return &geojson.FeatureCollection{Features: features}, nil
}
