package cyclinganalytics

import (
	"errors"
	"strconv"
	"time"

	"github.com/twpayne/go-gpx"

	"github.com/bzimmer/activity"
)

var _ activity.GPXEncoder = (*Ride)(nil)

func (r *Ride) GPX() (*gpx.GPX, error) {
	s := r.Streams
	if s.Latitude == nil || s.Longitude == nil {
		return nil, errors.New("required lat and lng not found")
	}

	// the streams are sampled together, one per second, but guard against a
	// short one rather than index past its end
	n := min(len(s.Latitude), len(s.Longitude))
	pts := make([]gpx.WptType, n)
	wpts := make([]*gpx.WptType, n)
	for i := range n {
		pts[i].Lat = s.Latitude[i]
		pts[i].Lon = s.Longitude[i]
		if i < len(s.Elevation) {
			pts[i].Ele = s.Elevation[i]
		}
		pts[i].Time = gpx.MToTime(float64(r.UTCDatetime.Add(time.Second * time.Duration(i)).Unix()))
		wpts[i] = &pts[i]
	}

	return &gpx.GPX{
		Creator: activity.UserAgent,
		Metadata: &gpx.MetadataType{
			Name: strconv.FormatInt(r.ID, 10),
		},
		Trk: []*gpx.TrkType{
			{
				Src:    _baseURL,
				TrkSeg: []*gpx.TrkSegType{{TrkPt: wpts}},
			},
		},
	}, nil
}
