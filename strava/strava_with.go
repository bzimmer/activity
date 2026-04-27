package strava

import (
	"context"
	"net/http"
	"time"

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
	c.base.Config.Endpoint = Endpoint()
	opts = append(opts, withServices())
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// WithConfig sets the underlying oauth2.Config.
func WithConfig(config oauth2.Config) Option {
	return func(c *Client) error {
		c.base.Config = config
		return nil
	}
}

// WithClientCredentials provides the client api credentials for the application.
func WithClientCredentials(clientID, clientSecret string) Option {
	return func(c *Client) error {
		c.base.Config.ClientID = clientID
		c.base.Config.ClientSecret = clientSecret
		return nil
	}
}

// WithAutoRefresh refreshes access tokens automatically.
// The order of this option matters because it is dependent on the client's
// config and token. Use this option after With*Credentials.
func WithAutoRefresh(ctx context.Context) Option {
	return func(c *Client) error {
		return httpclient.ApplyAutoRefresh(ctx, c.base)
	}
}

// WithToken sets the underlying oauth2.Token.
func WithToken(token *oauth2.Token) Option {
	return func(c *Client) error {
		c.base.Token = token
		return nil
	}
}

// WithTokenCredentials provides the tokens for an authenticated user.
func WithTokenCredentials(accessToken, refreshToken string, expiry time.Time) Option {
	return func(c *Client) error {
		c.base.Token.AccessToken = accessToken
		c.base.Token.RefreshToken = refreshToken
		c.base.Token.Expiry = expiry
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

func (c *Client) do(req *http.Request, v any) error {
	return c.base.Do(req, v)
}
