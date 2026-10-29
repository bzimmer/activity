package garmin

import (
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity/internal/httpclient"
)

type service struct {
	client *Client
}

// Option provides a configuration mechanism for a Client.
type Option func(*Client) error

// NewClient creates a new client and applies all provided Options.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		base: httpclient.New(func() *Fault { return &Fault{} }),
	}
	opts = append(opts, withServices())
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// WithToken sets the bearer token, in place of exchanging an API key.
func WithToken(token *oauth2.Token) Option {
	return func(c *Client) error {
		c.base.Token = token
		return nil
	}
}

// WithRateLimiter rate limits the client's api calls.
func WithRateLimiter(r *rate.Limiter) Option {
	return func(c *Client) error {
		return httpclient.ApplyRateLimiter(c.base, r)
	}
}

// WithHTTPTracing enables tracing http calls.
func WithHTTPTracing(debug bool) Option {
	return func(c *Client) error {
		return httpclient.ApplyHTTPTracing(c.base, debug)
	}
}

// WithTransport sets the underlying http client transport.
func WithTransport(t http.RoundTripper) Option {
	return func(c *Client) error {
		return httpclient.ApplyTransport(c.base, t)
	}
}

// WithHTTPClient sets the underlying http client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		return httpclient.ApplyHTTPClient(c.base, client)
	}
}
