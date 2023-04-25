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
