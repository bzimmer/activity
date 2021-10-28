package cyclinganalytics

//go:generate genwith --do --client --endpoint-func --config --token --ratelimit --package cyclinganalytics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/bzimmer/activity"
	"golang.org/x/oauth2"
)

const _baseURL = "https://www.cyclinganalytics.com/api"

// APIOption for configuring API requests
type APIOption func(url.Values) error

// Client for accessing Cycling Analytics' API
type Client struct {
	config  oauth2.Config
	token   *oauth2.Token
	client  *http.Client
	baseURL string

	User  *UserService
	Rides *RidesService
}

// Endpoint is CyclingAnalytics's OAuth 2.0 endpoint
func Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:   "https://www.cyclinganalytics.com/api/auth",
		TokenURL:  "https://www.cyclinganalytics.com/api/token",
		AuthStyle: oauth2.AuthStyleAutoDetect,
	}
}

func withServices() Option {
	return func(c *Client) error {
		c.User = &UserService{client: c}
		c.Rides = &RidesService{client: c}
		if c.baseURL == "" {
			c.baseURL = _baseURL
		}
		return nil
	}
}

// WithBaseURL specifies the base url
func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		c.baseURL = baseURL
		return nil
	}
}

func (c *Client) newAPIRequest(ctx context.Context, method, uri string, values *url.Values, body io.Reader) (*http.Request, error) {
	if c.token.AccessToken == "" {
		return nil, errors.New("accessToken required")
	}
	q := fmt.Sprintf("%s/%s", c.baseURL, uri)
	if values != nil {
		q = fmt.Sprintf("%s?%s", q, values.Encode())
	}
	u, err := url.Parse(q)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", activity.UserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.token.AccessToken))
	return req, nil
}

func (c *Client) Uploader() activity.Uploader {
	return newUploader(c.Rides)
}
