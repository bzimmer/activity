package hammerhead

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/internal/httpclient"
)

const (
	authURL = "https://api.hammerhead.io/v1/auth"
	apiURL  = "https://api.hammerhead.io/v1/api"
)

// Endpoint is Hammerhead's OAuth 2.0 endpoint
func Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:   authURL + "/oauth/authorize",
		TokenURL:  authURL + "/oauth/token",
		AuthStyle: oauth2.AuthStyleInParams,
	}
}

// Client for accessing Hammerhead's API
type Client struct {
	base    *httpclient.Client[*Fault]
	authURL string
	apiURL  string

	Activities *ActivitiesService
}

// Exporter returns an Exporter for this client
func (c *Client) Exporter() activity.Exporter {
	return c.Activities
}

func withServices() Option {
	return func(c *Client) error {
		c.Activities = &ActivitiesService{client: c}
		if c.authURL == "" {
			c.authURL = authURL
		}
		if c.apiURL == "" {
			c.apiURL = apiURL
		}
		return nil
	}
}

// WithAPIURL specifies the API base url
func WithAPIURL(apiURL string) Option {
	return func(c *Client) error {
		c.apiURL = apiURL
		return nil
	}
}

// WithAuthURL specifies the auth base url
func WithAuthURL(authURL string) Option {
	return func(c *Client) error {
		c.authURL = authURL
		return nil
	}
}

func (c *Client) newAPIRequest(ctx context.Context, method, uri string) (*http.Request, error) {
	if c.base.Token.AccessToken == "" {
		return nil, errors.New("accessToken required")
	}
	u, err := url.Parse(fmt.Sprintf("%s/%s", c.apiURL, uri))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", activity.UserAgent)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.base.Token.AccessToken))
	return req, nil
}

func (c *Client) do(req *http.Request, v any) error {
	return c.base.Do(req, v)
}
