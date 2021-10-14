package strava

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/bzimmer/activity"
	"golang.org/x/sync/errgroup"
)

// ActivityService is the API for activity endpoints
type ActivityService service

// ActivityIterFunc is called for each activity in the results
type ActivityIterFunc func(*Activity) (bool, error)

type channelPaginator struct {
	service    ActivityService
	count      int
	activities chan *ActivityResult
}

func (p *channelPaginator) PageSize() int {
	return PageSize
}

func (p *channelPaginator) Count() int {
	return p.count
}

func (p *channelPaginator) Do(ctx context.Context, spec activity.Pagination) (int, error) {
	uri := fmt.Sprintf("athlete/activities?page=%d&per_page=%d", spec.Start, spec.Count)
	req, err := p.service.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return 0, err
	}
	var acts []*Activity
	err = p.service.client.do(req, &acts)
	if err != nil {
		return 0, err
	}
	for _, act := range acts {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case p.activities <- &ActivityResult{Activity: act}:
			p.count++
		}
		if p.count == spec.Total {
			break
		}
	}
	return len(acts), nil
}

// Streams returns the activity's data streams
func (s *ActivityService) Streams(ctx context.Context, activityID int64, streams ...string) (*Streams, error) {
	if err := s.validateStreams(streams); err != nil {
		return nil, err
	}
	keys := strings.Join(streams, ",")
	uri := fmt.Sprintf("activities/%d/streams/%s?key_by_type=true", activityID, keys)
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	sts := &Streams{}
	err = s.client.do(req, sts)
	if err != nil {
		return nil, err
	}
	sts.ActivityID = activityID
	return sts, err
}

// Activity returns the activity specified by id
func (s *ActivityService) Activity(ctx context.Context, activityID int64, streams ...string) (*Activity, error) {
	if len(streams) > 0 {
		// confirm valid streams before querying strava for the activity
		if err := s.validateStreams(streams); err != nil {
			return nil, err
		}
	}
	uri := fmt.Sprintf("activities/%d", activityID)
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	var sms *Streams
	var act *Activity
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		act = &Activity{}
		return s.client.do(req, act)
	})
	if len(streams) > 0 {
		grp.Go(func() error {
			sms, err = s.Streams(ctx, activityID, streams...)
			return err
		})
	}
	if err := grp.Wait(); err != nil {
		return nil, err
	}
	act.Streams = sms
	return act, nil
}

// Activities returns a channel for activities and errors for an athlete
//
// Either the first error or last activity will close the channel
func (s *ActivityService) Activities(ctx context.Context, spec activity.Pagination) <-chan *ActivityResult {
	acts := make(chan *ActivityResult, PageSize)
	go func() {
		defer close(acts)
		p := &channelPaginator{service: *s, activities: acts}
		err := activity.Paginate(ctx, p, spec)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case acts <- &ActivityResult{Err: err}:
				return
			}
		}
	}()
	return acts
}

func (s *ActivityService) ActivitiesIter(ctx context.Context, spec activity.Pagination, iter ActivityIterFunc) error {
	for ar := range s.Activities(ctx, spec) {
		if ar.Err != nil {
			return ar.Err
		}
		ok, err := iter(ar.Activity)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	return nil
}

// Upload the file for the user
//
// More information can be found at https://developers.strava.com/docs/uploads/
func (s *ActivityService) Upload(ctx context.Context, file *activity.File) (*Upload, error) {
	if file == nil || file.Name == "" || file.Format == activity.FormatOriginal {
		return nil, errors.New("missing upload file, name, or format")
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if err := w.WriteField("filename", file.Name); err != nil {
		return nil, err
	}
	if err := w.WriteField("data_type", file.Format.String()); err != nil {
		return nil, err
	}
	fw, err := w.CreateFormFile("file", file.Name)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}

	req, err := s.client.newAPIRequest(ctx, http.MethodPost, "uploads", &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	res := &Upload{}
	err = s.client.do(req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Status returns the status of an upload request
//
// More information can be found at https://developers.strava.com/docs/uploads/
func (s *ActivityService) Status(ctx context.Context, uploadID int64) (*Upload, error) {
	uri := fmt.Sprintf("uploads/%d", uploadID)
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	res := &Upload{}
	err = s.client.do(req, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// https://developers.strava.com/docs/reference/#api-models-StreamSet
func streamsets() map[string]string {
	return map[string]string{
		"altitude":        "The sequence of altitude values for this stream, in meters [float]",
		"cadence":         "The sequence of cadence values for this stream, in rotations per minute [integer]",
		"distance":        "The sequence of distance values for this stream, in meters [float]",
		"grade_smooth":    "The sequence of grade values for this stream, as percents of a grade [float]",
		"heartrate":       "The sequence of heart rate values for this stream, in beats per minute [integer]",
		"latlng":          "The sequence of lat/long values for this stream [float, float]",
		"moving":          "The sequence of moving values for this stream, as boolean values [boolean]",
		"temp":            "The sequence of temperature values for this stream, in celsius degrees [float]",
		"time":            "The sequence of time values for this stream, in seconds [integer]",
		"velocity_smooth": "The sequence of velocity values for this stream, in meters per second [float]",
		"watts":           "The sequence of power values for this stream, in watts [integer]",
	}
}

// StreamSets returns the list of valid stream names
func (s *ActivityService) StreamSets() map[string]string {
	return streamsets()
}

func (s *ActivityService) validateStreams(streams []string) error {
	x := streamsets()
	for i := range streams {
		_, ok := x[streams[i]]
		if !ok {
			return fmt.Errorf("invalid stream '%s'", streams[i])
		}
	}
	return nil
}

// Photos returns the metadata (not the photo itself) for an activity
// Size can be (0, 64, 1024, 2048)
func (s *ActivityService) Photos(ctx context.Context, activityID int64, size int) ([]*Photo, error) {
	uri := fmt.Sprintf("activities/%d/photos?photo_sources=true&size=%d", activityID, size)
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	var photos []*Photo
	err = s.client.do(req, &photos)
	if err != nil {
		return nil, err
	}
	return photos, nil
}
