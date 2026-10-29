// Package garmin is a client for Garmin's Personal API: a key made in
// Garmin Connect (Settings, Personal API) is exchanged for a short-lived
// bearer token, which reads the activity list and each activity's original
// upload. See github.com/garmin/connect-personal-api-python for Garmin's
// reference client.
package garmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/internal/httpclient"
)

const apiURL = "https://personal.connectapi.garmin.com"

// Client for accessing Garmin's Personal API
type Client struct {
	base   *httpclient.Client[*Fault]
	apiURL string
	apiKey string

	Auth       *AuthService
	Activities *ActivitiesService
}

// Exporter returns an Exporter for this client
func (c *Client) Exporter() activity.Exporter {
	return c.Activities
}

func withServices() Option {
	return func(c *Client) error {
		c.Auth = &AuthService{client: c}
		c.Activities = &ActivitiesService{client: c}
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

// WithAPIKey provides the Personal API key, exchanged for a bearer token
// on the first request and again whenever the token expires.
func WithAPIKey(apiKey string) Option {
	return func(c *Client) error {
		c.apiKey = apiKey
		return nil
	}
}

// newAPIRequest returns a request carrying the bearer token, exchanging the
// API key for one first when there is none or it has expired.
func (c *Client) newAPIRequest(ctx context.Context, method, uri, accept string) (*http.Request, error) {
	if !c.base.Token.Valid() {
		if c.apiKey == "" {
			return nil, errors.New("apiKey or accessToken required")
		}
		if _, err := c.Auth.Token(ctx); err != nil {
			return nil, err
		}
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
	req.Header.Set("Accept", accept)
	req.Header.Set("Authorization", "Bearer "+c.base.Token.AccessToken)
	return req, nil
}

// send executes the request, returning the open response or a Fault for a
// status of 400 or more. Garmin's error bodies are not reliably JSON, so
// unlike httpclient.Client.Do an undecodable body still yields a Fault.
func (c *Client) send(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	res, err := c.base.HTTP.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, err
		}
	}
	if res.StatusCode < http.StatusBadRequest {
		return res, nil
	}
	defer res.Body.Close()
	f := &Fault{}
	_ = json.NewDecoder(res.Body).Decode(f)
	f.SetDefaults(res.StatusCode, http.StatusText(res.StatusCode))
	if secs, aerr := strconv.Atoi(res.Header.Get("Retry-After")); aerr == nil && secs > 0 {
		f.RetryAfter = time.Duration(secs) * time.Second
	}
	return nil, f
}

// do executes the request and decodes its JSON body into v.
func (c *Client) do(req *http.Request, v any) error {
	res, err := c.send(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err = json.NewDecoder(res.Body).Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
