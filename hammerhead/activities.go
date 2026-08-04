package hammerhead

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bzimmer/activity"
)

const pageSize = 100

// ActivitiesService provides access to Hammerhead activity endpoints
type ActivitiesService service

type activitiesPaginator struct {
	service    ActivitiesService
	activities []*ActivitySummary
	startDate  string
}

func (p *activitiesPaginator) PageSize() int {
	return pageSize
}

func (p *activitiesPaginator) Count() int {
	return len(p.activities)
}

func (p *activitiesPaginator) Do(ctx context.Context, spec activity.Pagination) (int, error) {
	v := url.Values{}
	v.Set("page", fmt.Sprintf("%d", spec.Start))
	v.Set("perPage", fmt.Sprintf("%d", spec.Count))
	if p.startDate != "" {
		v.Set("startDate", p.startDate)
	}
	req, err := p.service.client.newAPIRequest(ctx, http.MethodGet, "activities?"+v.Encode())
	if err != nil {
		return 0, err
	}
	res := &ActivitiesPage{}
	if err = p.service.client.do(req, res); err != nil {
		return 0, err
	}
	if spec.Total > 0 && len(p.activities)+len(res.Data) > spec.Total {
		res.Data = res.Data[:spec.Total-len(p.activities)]
	}
	p.activities = append(p.activities, res.Data...)
	if res.CurrentPage >= res.TotalPages {
		return 0, nil
	}
	return len(res.Data), nil
}

// Activities returns a slice of activity summaries
func (s *ActivitiesService) Activities(
	ctx context.Context, spec activity.Pagination, startDate string) ([]*ActivitySummary, error) {
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

// Activity returns a single activity by ID
func (s *ActivitiesService) Activity(ctx context.Context, activityID string) (*Activity, error) {
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, fmt.Sprintf("activities/%s", activityID))
	if err != nil {
		return nil, err
	}
	res := &Activity{}
	if err = s.client.do(req, res); err != nil {
		return nil, err
	}
	return res, nil
}

// File downloads the original FIT file for an activity
func (s *ActivitiesService) File(ctx context.Context, activityID string) (*activity.File, error) {
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, fmt.Sprintf("activities/%s/file", activityID))
	if err != nil {
		return nil, err
	}
	res, err := s.client.base.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= http.StatusBadRequest {
		defer res.Body.Close()
		f := &Fault{}
		f.SetDefaults(res.StatusCode, http.StatusText(res.StatusCode))
		return nil, f
	}
	return &activity.File{
		Reader:   res.Body,
		Filename: fmt.Sprintf("%s.fit", activityID),
		Name:     activityID,
		Format:   activity.FormatFIT,
	}, nil
}

// Export implements activity.Exporter by downloading the FIT file for an activity.
// The activityID is provided as an int64 but Hammerhead uses string IDs in their API;
// the int64 value is formatted as a decimal string for the request.
func (s *ActivitiesService) Export(ctx context.Context, activityID int64) (*activity.Export, error) {
	id := fmt.Sprintf("%d", activityID)
	f, err := s.File(ctx, id)
	if err != nil {
		return nil, err
	}
	return &activity.Export{File: f, ID: activityID}, nil
}

