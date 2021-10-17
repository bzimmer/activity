package rwgps

//go:generate genwith --do --client --token --config --ratelimit --package rwgps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"

	"github.com/bzimmer/activity"
)

const (
	apiVersion = "2"
	_baseURL   = "https://ridewithgps.com"
)

// Client for communicating with RWGPS
type Client struct {
	config  oauth2.Config
	token   *oauth2.Token
	client  *http.Client
	baseURL string

	Users *UsersService
	Trips *TripsService
}

func (c *Client) Uploader() activity.Uploader {
	return newUploader(c.Trips)
}

func withServices() Option {
	return func(c *Client) error {
		c.Users = &UsersService{client: c}
		c.Trips = &TripsService{client: c}
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

func (c *Client) newAPIRequest(ctx context.Context, uri string, params map[string]string) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s", c.baseURL, uri))
	if err != nil {
		return nil, err
	}
	x := map[string]string{
		"version":    apiVersion,
		"apikey":     c.config.ClientID,
		"auth_token": c.token.AccessToken,
	}
	for k, v := range params {
		x[k] = v
	}
	b, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", activity.UserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
