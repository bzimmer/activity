package garmin

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/bzimmer/activity"
)

const (
	pageSize     = 100
	searchURI    = "activitylist-service/activities/search/activities"
	downloadURI  = "download-service/files/activity/"
	acceptJSON   = "application/json"
	acceptOrigin = "application/zip, application/x-zip-compressed, application/octet-stream"
	// maxFileSize bounds an unzipped activity file; the largest FIT files
	// run to a few MB
	maxFileSize = 256 << 20
)

// ActivitiesService provides access to Garmin activity endpoints
type ActivitiesService service

type activitiesPaginator struct {
	service    ActivitiesService
	activities []*ActivitySummary
	startDate  time.Time
}

func (p *activitiesPaginator) PageSize() int {
	return pageSize
}

func (p *activitiesPaginator) Count() int {
	return len(p.activities)
}

// Do reads one page; Garmin pages by offset, not page number.
func (p *activitiesPaginator) Do(ctx context.Context, spec activity.Pagination) (int, error) {
	v := url.Values{}
	v.Set("start", strconv.Itoa((spec.Start-1)*spec.Count))
	v.Set("limit", strconv.Itoa(spec.Count))
	if !p.startDate.IsZero() {
		v.Set("startDate", p.startDate.Format(time.DateOnly))
	}
	req, err := p.service.client.newAPIRequest(ctx, http.MethodGet, searchURI+"?"+v.Encode(), acceptJSON)
	if err != nil {
		return 0, err
	}
	var page []*ActivitySummary
	if err = p.service.client.do(req, &page); err != nil {
		return 0, err
	}
	n := len(page)
	if spec.Total > 0 && len(p.activities)+len(page) > spec.Total {
		page = page[:spec.Total-len(p.activities)]
	}
	p.activities = append(p.activities, page...)
	if n < spec.Count {
		// a short page is the last one
		return 0, nil
	}
	return n, nil
}

// Activities returns the summaries of activities started on or after
// startDate (every activity when it is zero), most recent first.
func (s *ActivitiesService) Activities(
	ctx context.Context, spec activity.Pagination, startDate time.Time) ([]*ActivitySummary, error) {
	p := &activitiesPaginator{
		service:    *s,
		startDate:  startDate,
		activities: make([]*ActivitySummary, 0),
	}
	if err := activity.Paginate(ctx, p, spec); err != nil {
		return nil, err
	}
	return p.activities, nil
}

// File downloads an activity's original upload, which Garmin serves as a
// ZIP holding the one file the device uploaded, and returns that file:
// normally FIT, though an older upload can be TCX or GPX. Garmin answers
// 404 for a manual entry, which has none, and 403 for an activity from a
// watch older than FIT, which Connect only exports as TCX or GPX.
func (s *ActivitiesService) File(ctx context.Context, activityID int64) (*activity.File, error) {
	id := strconv.FormatInt(activityID, 10)
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, downloadURI+id, acceptOrigin)
	if err != nil {
		return nil, err
	}
	res, err := s.client.send(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	name, data, err := unzipOne(body)
	if err != nil {
		return nil, fmt.Errorf("activity %s: %w", id, err)
	}
	return &activity.File{
		Reader:   bytes.NewReader(data),
		Filename: name,
		Name:     id,
		Format:   activity.ToFormat(path.Ext(name)),
	}, nil
}

// unzipOne returns the base name and contents of the only file in a ZIP.
func unzipOne(body []byte) (string, []byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", nil, fmt.Errorf("reading the original upload: %w", err)
	}
	var files []*zip.File
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			files = append(files, f)
		}
	}
	if len(files) != 1 {
		return "", nil, fmt.Errorf("expected one file in the original upload, found %d", len(files))
	}
	rc, err := files[0].Open()
	if err != nil {
		return "", nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxFileSize+1))
	if err != nil {
		return "", nil, err
	}
	if len(data) > maxFileSize {
		return "", nil, fmt.Errorf("%s is larger than %d bytes", files[0].Name, maxFileSize)
	}
	return path.Base(files[0].Name), data, nil
}

// Export implements activity.Exporter by downloading the file of an activity's original upload.
func (s *ActivitiesService) Export(ctx context.Context, activityID int64) (*activity.Export, error) {
	f, err := s.File(ctx, activityID)
	if err != nil {
		return nil, err
	}
	return &activity.Export{File: f, ID: activityID}, nil
}
