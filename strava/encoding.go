package strava

import (
	"errors"
	"fmt"
	"time"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-gpx"
	"github.com/twpayne/go-polyline"

	"github.com/bzimmer/activity"
)

var _ activity.GPXEncoder = (*Route)(nil)
var _ activity.GPXEncoder = (*Activity)(nil)

func polylineToLineString(polylines ...string) (*geom.LineString, error) {
	const N = 2
	var coords []float64
	var linestring *geom.LineString
	for _, p := range polylines {
		if p == "" {
			continue
		}
		c, _, err := polyline.DecodeCoords([]byte(p))
		if err != nil {
			return nil, err
		}
		coords = make([]float64, len(c)*N)
		for i := 0; i < len(c); i++ {
			x := N * i
			coords[x+0] = c[i][1]
			coords[x+1] = c[i][0]
		}
		return geom.NewLineStringFlat(geom.XY, coords), nil
	}
	if linestring == nil {
		return nil, errors.New("no valid polyline")
	}
	return linestring, nil
}

// GPX representation of an activity
func (a *Activity) GPX() (*gpx.GPX, error) {
	if a.Streams == nil {
		ls, err := a.Map.LineString()
		if err != nil {
			return nil, err
		}
		mls := geom.NewMultiLineString(ls.Layout())
		err = mls.Push(ls)
		if err != nil {
			return nil, err
		}
		trk := gpx.NewTrkType(mls)
		trk.Name = a.Name
		trk.Desc = a.Description
		trk.Link = []*gpx.LinkType{
			{
				HREF: fmt.Sprintf("https://strava.com/activities/%d", a.ID),
			},
		}
		x := &gpx.GPX{
			Trk: []*gpx.TrkType{trk},
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
	rte := gpx.NewRteType(ls)
	rte.Name = r.Name
	rte.Desc = r.Description
	rte.Link = []*gpx.LinkType{
		{
			HREF: fmt.Sprintf("https://strava.com/routes/%d", r.ID),
		},
	}
	x := &gpx.GPX{
		Rte: []*gpx.RteType{rte},
	}
	return x, nil
}

func (a *Activity) toGPXFromStreams() (*gpx.GPX, error) {
	if a.Streams == nil {
		return nil, errors.New("no streams available for gpx encoding")
	}
	if a.Streams.LatLng == nil || a.Streams.Time == nil {
		return nil, errors.New("both latlng and time streams are required for gpx encoding")
	}
	points := make([]*gpx.WptType, len(a.Streams.Time.Data))
	for i := range a.Streams.Time.Data {
		points[i] = &gpx.WptType{
			Lat:  a.Streams.LatLng.Data[i][0],
			Lon:  a.Streams.LatLng.Data[i][1],
			Time: a.StartDate.Add(time.Duration(a.Streams.Time.Data[i]) * time.Second),
		}
		if a.Streams.Elevation != nil {
			points[i].Ele = a.Streams.Elevation.Data[i].Meters()
		}
	}
	x := &gpx.GPX{
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
