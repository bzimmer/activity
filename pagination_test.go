package activity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
)

type paginator struct {
	err        bool
	total      int
	count      int
	page       int
	called     int
	iterations int
}

func (p *paginator) PageSize() int {
	return p.page
}

func (p *paginator) Count() int {
	return p.total
}

func (p *paginator) Do(_ context.Context, _ activity.Pagination) (int, error) {
	if p.err {
		return 0, errors.New("error in paginator.do")
	}
	p.called++
	p.total += p.count
	return p.count, nil
}

func TestPaginate(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name       string
		err        string
		pagination activity.Pagination
		paginator  *paginator
	}{
		{
			name: "error in paginator.do",
			err:  "error in paginator.do",
			pagination: activity.Pagination{
				Total: 100, Count: 10, Start: 1,
			},
			paginator: &paginator{err: true},
		},
		{
			name: "total less than zero",
			err:  "total less than zero",
			pagination: activity.Pagination{
				Total: -1, Count: 10, Start: 1,
			},
			paginator: &paginator{err: true},
		},
		{
			name: "simple",
			pagination: activity.Pagination{
				Total: 100, Count: 10, Start: 1,
			},
			paginator: &paginator{iterations: 1},
		},
		{
			name: "optimize",
			pagination: activity.Pagination{
				Total: 10, Count: 100, Start: 1,
			},
			paginator: &paginator{page: 100, total: 10, count: 100, iterations: 1},
		},
		{
			name: "total <= count",
			pagination: activity.Pagination{
				Total: 10, Count: 100, Start: 1,
			},
			paginator: &paginator{total: 10, count: 100, iterations: 1},
		},
		{
			name: "start == 0 && count <= 0",
			pagination: activity.Pagination{
				Total: 10, Count: 0, Start: 0,
			},
			paginator: &paginator{total: 10, count: 100, iterations: 1},
		},
		{
			name: "two iterations",
			pagination: activity.Pagination{
				Total: 10, Count: 0, Start: 0,
			},
			paginator: &paginator{total: 5, count: 3, iterations: 2},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := activity.Paginate(context.Background(), tt.paginator, tt.pagination)
			if tt.err != "" {
				a.Error(err)
				a.Contains(err.Error(), tt.err)
				return
			}
			a.NoError(err)
			a.Equal(tt.paginator.iterations, tt.paginator.called)
		})
	}
}

// specPaginator records the specs it is asked for and returns a full page
type specPaginator struct {
	specs []activity.Pagination
	count int
}

func (p *specPaginator) PageSize() int {
	return 100
}

func (p *specPaginator) Count() int {
	return p.count
}

func (p *specPaginator) Do(_ context.Context, spec activity.Pagination) (int, error) {
	p.specs = append(p.specs, spec)
	p.count += spec.Count
	return spec.Count, nil
}

func TestPaginateSpec(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name       string
		pagination activity.Pagination
		specs      []activity.Pagination
	}{
		{
			name:       "one short page from the beginning",
			pagination: activity.Pagination{Total: 50},
			specs:      []activity.Pagination{{Total: 50, Start: 1, Count: 50}},
		},
		{
			name:       "total under the requested count from the beginning",
			pagination: activity.Pagination{Total: 5, Count: 10, Start: 1},
			specs:      []activity.Pagination{{Total: 5, Start: 1, Count: 5}},
		},
		{
			// page three of the default size, not of fifty
			name:       "short total past the first page keeps the page size",
			pagination: activity.Pagination{Total: 50, Start: 3},
			specs:      []activity.Pagination{{Total: 50, Start: 3, Count: 100}},
		},
		{
			// page two of ten, not of five
			name:       "total under the requested count past the first page",
			pagination: activity.Pagination{Total: 5, Count: 10, Start: 2},
			specs:      []activity.Pagination{{Total: 5, Start: 2, Count: 10}},
		},
		{
			name:       "requested count across pages",
			pagination: activity.Pagination{Total: 200, Count: 100, Start: 2},
			specs: []activity.Pagination{
				{Total: 200, Start: 2, Count: 100},
				{Total: 200, Start: 3, Count: 100},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &specPaginator{}
			a.NoError(activity.Paginate(context.Background(), p, tt.pagination))
			a.Equal(tt.specs, p.specs)
		})
	}
}
