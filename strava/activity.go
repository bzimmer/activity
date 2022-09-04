package strava

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/bzimmer/activity"
	"golang.org/x/sync/errgroup"
)

// ActivityService is the API for activity endpoints
type ActivityService service

// ActivityIterFunc is called for each activity in the results
type ActivityIterFunc func(*Activity) (bool, error)

// fileNameRE allowable characters
var fileNameRE = regexp.MustCompile("[A-Za-z0-9-]+")

// WithDateRange sets the before and after date range
func WithDateRange(before, after time.Time) APIOption {
	return func(v url.Values) error {
		if !before.IsZero() && !after.IsZero() {
			if after.After(before) {
				return errors.New("invalid date range")
			}
		}
		if !before.IsZero() {
			v.Set("before", fmt.Sprintf("%d", before.Unix()))
		}
		if !after.IsZero() {
			v.Set("after", fmt.Sprintf("%d", after.Unix()))
		}
		return nil
	}
}

type channelPaginator struct {
	count      int
	options    []APIOption
	service    ActivityService
	activities chan *ActivityResult
}

func (p *channelPaginator) PageSize() int {
	return PageSize
}

func (p *channelPaginator) Count() int {
	return p.count
}

func (p *channelPaginator) Do(ctx context.Context, spec activity.Pagination) (int, error) {
	v := make(url.Values)
	v.Set("page", fmt.Sprintf("%d", spec.Start))
	v.Set("per_page", fmt.Sprintf("%d", spec.Count))
	for _, opt := range p.options {
		if opt == nil {
			continue
		}
		if err := opt(v); err != nil {
			return 0, err
		}
	}
	uri := fmt.Sprintf("athlete/activities?%s", v.Encode())
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
	if err = s.client.do(req, sts); err != nil {
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
	if err = grp.Wait(); err != nil {
		return nil, err
	}
	act.Streams = sms
	return act, nil
}

// Activities returns a channel for activities and errors for an athlete
func (s *ActivityService) Activities(ctx context.Context, spec activity.Pagination, opts ...APIOption) <-chan *ActivityResult {
	acts := make(chan *ActivityResult, PageSize)
	go func() {
		defer close(acts)
		p := &channelPaginator{service: *s, activities: acts, options: opts}
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

// ActivitiesIter executes the iter function over the results of the channel
func ActivitiesIter(res <-chan *ActivityResult, iter ActivityIterFunc) error {
	for ar := range res {
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
	if err = s.client.do(req, res); err != nil {
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
	if err = s.client.do(req, &photos); err != nil {
		return nil, err
	}
	return photos, nil
}

// Export exports an activity in the GPX format
func (s *ActivityService) Export(ctx context.Context, activityID int64) (*activity.Export, error) {
	act, err := s.Activity(ctx, activityID, "latlng", "time", "altitude")
	if err != nil {
		return nil, err
	}
	x, err := act.GPX()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err = x.Write(&buf); err != nil {
		return nil, err
	}
	name := strings.Join(fileNameRE.FindAllString(act.Name, -1), "_")
	exp := &activity.Export{
		File: &activity.File{
			Reader: &buf,
			Name:   name,
			Format: activity.FormatGPX,
		},
		ID: activityID,
	}
	return exp, nil
}

// Update the given activity owned by the authenticated athlete
func (s *ActivityService) Update(ctx context.Context, act *UpdatableActivity) (*Activity, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(act); err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("activities/%d", act.ID)
	req, err := s.client.newAPIRequest(ctx, http.MethodPut, uri, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	res := new(Activity)
	if err = s.client.do(req, res); err != nil {
		return nil, err
	}
	return res, nil
}
