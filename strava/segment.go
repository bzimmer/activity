package strava

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bzimmer/activity"
)

// SegmentService is the API for segment effort endpoints.
type SegmentService service

type segmentPaginator struct {
	segmentEfforts []*SegmentEffort
	service        SegmentService
}

func (p *segmentPaginator) PageSize() int {
	return PageSize
}

func (p *segmentPaginator) Count() int {
	return len(p.segmentEfforts)
}

func (p *segmentPaginator) Do(ctx context.Context, spec activity.Pagination) (int, error) {
	uri := fmt.Sprintf("segment_efforts?page=%d&per_page=%d", spec.Start, spec.Count)
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
func (s *SegmentService) SegmentEfforts(ctx context.Context, spec activity.Pagination) ([]*SegmentEffort, error) {
	p := &segmentPaginator{service: *s, segmentEfforts: make([]*SegmentEffort, 0)}
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
