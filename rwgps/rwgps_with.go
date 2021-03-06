// Code generated by "genwith --do --client --token --config --ratelimit --package rwgps"; DO NOT EDIT.

package rwgps

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/bzimmer/httpwares"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

type service struct {
	client *Client //nolint:golint,structcheck
}

// Option provides a configuration mechanism for a Client
type Option func(*Client) error

// NewClient creates a new client and applies all provided Options
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		client: &http.Client{},
		token:  &oauth2.Token{},
		config: oauth2.Config{},
	}
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
		c.config = config
		return nil
	}
}

// WithAPICredentials provides the client api credentials for the application.
func WithClientCredentials(clientID, clientSecret string) Option {
	return func(c *Client) error {
		c.config.ClientID = clientID
		c.config.ClientSecret = clientSecret
		return nil
	}
}

// WithToken sets the underlying oauth2.Token.
func WithToken(token *oauth2.Token) Option {
	return func(c *Client) error {
		c.token = token
		return nil
	}
}

// WithTokenCredentials provides the tokens for an authenticated user.
func WithTokenCredentials(accessToken, refreshToken string, expiry time.Time) Option {
	return func(c *Client) error {
		c.token.AccessToken = accessToken
		c.token.RefreshToken = refreshToken
		c.token.Expiry = expiry
		return nil
	}
}

// WithRateLimiter rate limits the client's api calls
func WithRateLimiter(r *rate.Limiter) Option {
	return func(c *Client) error {
		if r == nil {
			return errors.New("nil limiter")
		}
		c.client.Transport = &httpwares.RateLimitTransport{
			Limiter:   r,
			Transport: c.client.Transport,
		}
		return nil
	}
}

// WithHTTPTracing enables tracing http calls.
func WithHTTPTracing(debug bool) Option {
	return func(c *Client) error {
		if !debug {
			return nil
		}
		c.client.Transport = &httpwares.VerboseTransport{
			Transport: c.client.Transport,
		}
		return nil
	}
}

// WithTransport sets the underlying http client transport.
func WithTransport(t http.RoundTripper) Option {
	return func(c *Client) error {
		if t == nil {
			return errors.New("nil transport")
		}
		c.client.Transport = t
		return nil
	}
}

// WithHTTPClient sets the underlying http client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		if client == nil {
			return errors.New("nil client")
		}
		c.client = client
		return nil
	}
}

// do executes the http request and populates v with the result.
func (c *Client) do(req *http.Request, v interface{}) error {
	ctx := req.Context()
	res, err := c.client.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return err
		}
	}
	defer res.Body.Close()

	httpError := res.StatusCode >= http.StatusBadRequest

	var obj interface{}
	if httpError {
		obj = &Fault{}
	} else {
		obj = v
	}

	if obj != nil {
		err := json.NewDecoder(res.Body).Decode(obj)
		if err == io.EOF {
			err = nil // ignore EOF errors caused by empty response body
		}
		if httpError {
			switch q := obj.(type) {
			case *Fault:
				if q.Code == 0 {
					q.Code = res.StatusCode
				}
				if q.Message == "" {
					q.Message = http.StatusText(res.StatusCode)
				}
				return q
			case error:
				return q
			default:
				return q.(error)
			}
		}
		return err
	}

	return nil
}
