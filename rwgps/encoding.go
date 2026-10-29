package rwgps

import (
	"strconv"

	"github.com/twpayne/go-gpx"

	"github.com/bzimmer/activity"
)

var _ activity.GPXEncoder = (*Trip)(nil)

func (t *Trip) GPX() (*gpx.GPX, error) {
	x := &gpx.GPX{
		Creator: activity.UserAgent,
		Metadata: &gpx.MetadataType{
			Name: strconv.FormatInt(t.ID, 10),
		},
	}
	switch t.Type {
	case TypeTrip.String():
		x.Trk = []*gpx.TrkType{{TrkSeg: []*gpx.TrkSegType{{TrkPt: t.waypoints(true)}}}}
	case TypeRoute.String():
		// routes do not have a `time` dimension
		x.Rte = []*gpx.RteType{{RtePt: t.waypoints(false)}}
	}
	return x, nil
}

// waypoints converts the track points to waypoints backed by a single
// allocation rather than one per point
func (t *Trip) waypoints(timed bool) []*gpx.WptType {
	pts := make([]gpx.WptType, 0, len(t.TrackPoints))
	for _, tp := range t.TrackPoints {
		if tp == nil {
			continue
		}
		pt := gpx.WptType{Lat: tp.Latitude, Lon: tp.Longitude, Ele: tp.Elevation.Meters()}
		if timed {
			pt.Time = gpx.MToTime(tp.Time)
		}
		pts = append(pts, pt)
	}
	wpts := make([]*gpx.WptType, len(pts))
	for i := range pts {
		wpts[i] = &pts[i]
	}
	return wpts
}
