package strava

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bzimmer/activity"
)

// SegmentService is the API for segment endpoints.
type SegmentService service

type segmentPaginator struct {
	segments []*Segment
	service  SegmentService
}

func (p *segmentPaginator) PageSize() int {
	return PageSize
}

func (p *segmentPaginator) Count() int {
	return len(p.segments)
}

func (p *segmentPaginator) Do(ctx context.Context, spec activity.Pagination) (int, error) {
	uri := fmt.Sprintf("segments/starred?page=%d&per_page=%d", spec.Start, spec.Count)
	req, err := p.service.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return 0, err
	}
	var segs []*Segment
	err = p.service.client.do(req, &segs)
	if err != nil {
		return 0, err
	}
	if spec.Total > 0 && len(p.segments)+len(segs) > spec.Total {
		segs = segs[:spec.Total-len(p.segments)]
	}
	p.segments = append(p.segments, segs...)
	return len(segs), nil
}

// Segments returns a page of starred segments for the authenticated athlete.
func (s *SegmentService) Segments(ctx context.Context, spec activity.Pagination) ([]*Segment, error) {
	p := &segmentPaginator{service: *s, segments: make([]*Segment, 0)}
	err := activity.Paginate(ctx, p, spec)
	if err != nil {
		return nil, err
	}
	return p.segments, nil
}

// Segment returns a segment.
func (s *SegmentService) Segment(ctx context.Context, segmentID int64) (*Segment, error) {
	uri := fmt.Sprintf("segments/%d", segmentID)
	req, err := s.client.newAPIRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	seg := &Segment{}
	err = s.client.do(req, &seg)
	if err != nil {
		return nil, err
	}
	return seg, nil
}
