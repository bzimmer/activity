package strava

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bzimmer/activity"
)

// SegmentService is the API for segment effort endpoints.
type SegmentService service

// SegmentEffortIterFunc is called for each segment effort in the results.
type SegmentEffortIterFunc func(*SegmentEffort) (bool, error)

type segmentPaginator struct {
	segmentEfforts []*SegmentEffort
	service        SegmentService
	options        []APIOption
}

func (p *segmentPaginator) PageSize() int {
	return PageSize
}

func (p *segmentPaginator) Count() int {
	return len(p.segmentEfforts)
}

func (p *segmentPaginator) Do(ctx context.Context, spec activity.Pagination) (int, error) {
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
	uri := fmt.Sprintf("segment_efforts?%s", v.Encode())
	req, err := p.service.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return 0, err
	}
	var segs []*SegmentEffort
	err = p.service.client.do(req, &segs)
	if err != nil {
		return 0, err
	}
	if spec.Total > 0 && len(p.segmentEfforts)+len(segs) > spec.Total {
		segs = segs[:spec.Total-len(p.segmentEfforts)]
	}
	p.segmentEfforts = append(p.segmentEfforts, segs...)
	return len(segs), nil
}

// SegmentEfforts returns a page of segment efforts for the authenticated athlete.
//
// The returned segment efforts can be filtered by date using WithDateRange(before, after).
func (s *SegmentService) SegmentEfforts(
	ctx context.Context, spec activity.Pagination, opts ...APIOption) ([]*SegmentEffort, error) {
	p := &segmentPaginator{service: *s, segmentEfforts: make([]*SegmentEffort, 0), options: opts}
	err := activity.Paginate(ctx, p, spec)
	if err != nil {
		return nil, err
	}
	return p.segmentEfforts, nil
}

// SegmentEffort returns a segment effort.
func (s *SegmentService) SegmentEffort(ctx context.Context, segmentEffortID int64) (*SegmentEffort, error) {
	uri := fmt.Sprintf("segment_efforts/%d", segmentEffortID)
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	seg := &SegmentEffort{}
	err = s.client.do(req, &seg)
	if err != nil {
		return nil, err
	}
	return seg, nil
}

// SegmentEffortsIter executes the iter function over segment effort results.
func SegmentEffortsIter(segmentEfforts []*SegmentEffort, iter SegmentEffortIterFunc) error {
	for _, segmentEffort := range segmentEfforts {
		ok, err := iter(segmentEffort)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	return nil
}
