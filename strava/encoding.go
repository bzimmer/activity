package strava

import (
	"errors"
	"fmt"
	"time"

	"github.com/martinlindhe/unit"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-gpx"
	"github.com/twpayne/go-polyline"

	"github.com/bzimmer/activity"
)

var _ activity.GPXEncoder = (*Route)(nil)
var _ activity.GPXEncoder = (*Activity)(nil)

const gpxVersion = "1.1"

func polylineToLineString(polylines ...string) (*geom.LineString, error) {
	// the codec Strava uses, the go-polyline default
	codec := polyline.Codec{Dim: 2, Scale: 1e5}
	for _, p := range polylines {
		if p == "" {
			continue
		}
		// each value takes at least one byte, usually two to four, so half the
		// polyline's length is a close capacity guess that rarely needs to grow
		coords, _, err := codec.DecodeFlatCoords(make([]float64, 0, len(p)/2), []byte(p))
		if err != nil {
			return nil, err
		}
		// polylines are [lat, lng] while geom.XY is [lng, lat]
		for i := 0; i+1 < len(coords); i += 2 {
			coords[i], coords[i+1] = coords[i+1], coords[i]
		}
		return geom.NewLineStringFlat(geom.XY, coords), nil
	}
	return nil, errors.New("no valid polyline")
}

// GPX representation of an activity
func (a *Activity) GPX() (*gpx.GPX, error) {
	if a.Streams == nil {
		ls, err := a.Map.LineString()
		if err != nil {
			return nil, err
		}
		x := &gpx.GPX{
			Version: gpxVersion,
			Trk: []*gpx.TrkType{
				{
					Name:   a.Name,
					Desc:   a.Description,
					Link:   []*gpx.LinkType{{HREF: fmt.Sprintf("https://strava.com/activities/%d", a.ID)}},
					TrkSeg: []*gpx.TrkSegType{{TrkPt: waypoints(ls)}},
				},
			},
		}
		return x, nil
	}
	return a.toGPXFromStreams()
}

// GPX representation of a route
func (r *Route) GPX() (*gpx.GPX, error) {
	ls, err := r.Map.LineString()
	if err != nil {
		return nil, err
	}
	x := &gpx.GPX{
		Version: gpxVersion,
		Rte: []*gpx.RteType{
			{
				Name:  r.Name,
				Desc:  r.Description,
				Link:  []*gpx.LinkType{{HREF: fmt.Sprintf("https://strava.com/routes/%d", r.ID)}},
				RtePt: waypoints(ls),
			},
		},
	}
	return x, nil
}

// waypoints converts an XY line string to waypoints backed by a single
// allocation rather than one per point
func waypoints(ls *geom.LineString) []*gpx.WptType {
	coords := ls.FlatCoords()
	stride := ls.Stride()
	n := ls.NumCoords()
	pts := make([]gpx.WptType, n)
	wpts := make([]*gpx.WptType, n)
	for i := range n {
		x := stride * i
		pts[i].Lat = coords[x+1]
		pts[i].Lon = coords[x+0]
		wpts[i] = &pts[i]
	}
	return wpts
}

func (a *Activity) toGPXFromStreams() (*gpx.GPX, error) {
	if a.Streams == nil {
		return nil, errors.New("no streams available for gpx encoding")
	}
	if a.Streams.LatLng == nil || a.Streams.Time == nil {
		return nil, errors.New("both latlng and time streams are required for gpx encoding")
	}
	// the streams are sampled together but guard against a short one rather
	// than index past its end
	latlng, times := a.Streams.LatLng.Data, a.Streams.Time.Data
	var elevation []unit.Length
	if a.Streams.Elevation != nil {
		elevation = a.Streams.Elevation.Data
	}
	n := min(len(latlng), len(times))
	pts := make([]gpx.WptType, 0, n)
	for i := range n {
		// a sample without both coordinates has no place on the track
		if len(latlng[i]) < 2 {
			continue
		}
		pt := gpx.WptType{
			Lat:  latlng[i][0],
			Lon:  latlng[i][1],
			Time: a.StartDate.Add(time.Duration(times[i]) * time.Second),
		}
		if i < len(elevation) {
			pt.Ele = elevation[i].Meters()
		}
		pts = append(pts, pt)
	}
	points := make([]*gpx.WptType, len(pts))
	for i := range pts {
		points[i] = &pts[i]
	}
	x := &gpx.GPX{
		Version: gpxVersion,
		Trk: []*gpx.TrkType{
			{
				Name: a.Name,
				Desc: a.Description,
				Link: []*gpx.LinkType{
					{
						HREF: fmt.Sprintf("https://strava.com/activities/%d", a.ID),
					},
				},
				TrkSeg: []*gpx.TrkSegType{
					{
						TrkPt: points,
					},
				},
			},
		},
	}
	return x, nil
}
