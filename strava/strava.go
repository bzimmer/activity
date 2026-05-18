package strava

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/internal/httpclient"
)

const (
	_baseURL = "https://www.strava.com/api/v3"
	// PageSize default for querying bulk entities (eg activities, routes)
	PageSize = 100
)

// APIOption for configuring API requests
type APIOption func(url.Values) error

// WithDateRange sets the before and after date range.
func WithDateRange(before, after time.Time) APIOption {
	return func(v url.Values) error {
		if !before.IsZero() && !after.IsZero() {
			if after.After(before) {
				return errors.New("invalid date range")
			}
		}
		if !before.IsZero() {
			v.Set("before", fmt.Sprintf("%d", before.Unix()))
		}
		if !after.IsZero() {
			v.Set("after", fmt.Sprintf("%d", after.Unix()))
		}
		return nil
	}
}

// Endpoint is Strava's OAuth 2.0 endpoint
func Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{ //nolint:gosec // not a secret
		AuthURL:   "https://www.strava.com/oauth/authorize",
		TokenURL:  "https://www.strava.com/oauth/token",
		AuthStyle: oauth2.AuthStyleAutoDetect,
	}
}

// Client for accessing Strava's API
type Client struct {
	base    *httpclient.Client[*Fault]
	baseURL string

	Auth     *AuthService
	Route    *RouteService
	Segment  *SegmentService
	Webhook  *WebhookService
	Athlete  *AthleteService
	Activity *ActivityService
}

// Uploader returns an Uploader for this client
func (c *Client) Uploader() activity.Uploader {
	return newUploader(c.Activity)
}

// Exporter returns an Exporter for this client
func (c *Client) Exporter() activity.Exporter {
	return c.Activity
}

// WithBaseURL specifies the base url
func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		c.baseURL = baseURL
		return nil
	}
}

func withServices() Option {
	return func(c *Client) error {
		c.Auth = &AuthService{client: c}
		c.Route = &RouteService{client: c}
		c.Segment = &SegmentService{client: c}
		c.Webhook = &WebhookService{client: c}
		c.Athlete = &AthleteService{client: c}
		c.Activity = &ActivityService{client: c}
		if c.baseURL == "" {
			c.baseURL = _baseURL
		}
		return nil
	}
}

func (c *Client) newAPIRequest(ctx context.Context, method, uri string, body io.Reader) (*http.Request, error) {
	if c.base.Token.AccessToken == "" {
		return nil, errors.New("accessToken required")
	}
	u, err := url.Parse(fmt.Sprintf("%s/%s", c.baseURL, uri))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", activity.UserAgent)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.base.Token.AccessToken))
	return req, nil
}

func (c *Client) newWebhookRequest(
	ctx context.Context, method, uri string, body map[string]string) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s", c.baseURL, uri))
	if err != nil {
		return nil, err
	}
	var buf io.Reader
	if body != nil {
		form := url.Values{}
		form.Set("client_id", c.base.Config.ClientID)
		form.Set("client_secret", c.base.Config.ClientSecret)
		for key, value := range body {
			form.Set(key, value)
		}
		buf = io.NopCloser(bytes.NewBufferString(form.Encode()))
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded; param=value")
	}
	return req, nil
}
