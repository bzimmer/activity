package inreach

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// FeedService returns a feed of tracking points
type FeedService service

// Feed returns the feed for the user in the (optionally) specified time range
func (s *FeedService) Feed(ctx context.Context, user string, opts ...APIOption) (*Feed, error) {
	v := make(url.Values)
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(v); err != nil {
			return nil, err
		}
	}
	uri := fmt.Sprintf("Feed/Share/%s?%s", user, v.Encode())
	req, err := s.client.newRequest(ctx, http.MethodGet, uri)
	if err != nil {
		return nil, err
	}
	var feed Feed
	err = s.client.do(req, &feed)
	if err != nil {
		return nil, err
	}
	return &feed, nil
}
