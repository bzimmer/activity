package activity

import (
	"context"
	"errors"
)

// Pagination specifies how to paginate through resources
type Pagination struct {
	// Total number of resources to query
	Total int
	// Start querying at this page
	Start int
	// Count of the number of resources to query per page
	Count int
}

// Paginator paginates through results
type Paginator interface {
	// PageSize returns the number of resources to query per request
	PageSize() int
	// Count of the aggregate total of resources queried
	Count() int
	// Do executes the query using the pagination specification returning
	// the number of resources returned in this request or an error
	Do(ctx context.Context, spec Pagination) (int, error)
}

func Paginate(ctx context.Context, paginator Paginator, spec Pagination) error {
	var (
		start = spec.Start
		count = spec.Count
		total = spec.Total
	)
	if total < 0 {
		return errors.New("total less than zero")
	}
	if start <= 0 {
		start = 1
	}
	if count <= 0 {
		count = paginator.PageSize()
	}
	// if the request fits in one page from the beginning then fetch only what
	// is needed; past the first page the count stays as requested because
	// `start` counts pages of that size
	if total > 0 && start <= 1 && (total <= count || total < paginator.PageSize()) {
		count = total
	}
	return do(ctx, paginator, Pagination{Total: total, Start: start, Count: count})
}

func do(ctx context.Context, paginator Paginator, spec Pagination) error {
	for {
		n, err := paginator.Do(ctx, spec)
		if err != nil {
			return err
		}
		if n == 0 {
			// fewer than requested results is a possible scenario so break only if
			//  0 results were returned or we have enough to fulfill the request
			break
		}
		// @warning(bzimmer)
		// the `spec.Count` value must be consistent throughout the entire pagination
		all := paginator.Count()
		if spec.Total > 0 && all >= spec.Total {
			break
		}
		spec.Start++
	}
	return nil
}
