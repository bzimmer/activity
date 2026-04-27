// Package httpclient provides a generic HTTP client base shared by all API client packages.
package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bzimmer/httpwares"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// Fault is the interface constraint for HTTP API error types.
// Implementations are decoded from JSON error responses; SetDefaults populates
// the Code and Message fields from the HTTP status when the body does not supply them.
type Fault interface {
	error
	SetDefaults(code int, message string)
}

// Client holds the common HTTP state shared by all API clients.
// F is the package-specific error type used when an HTTP response indicates failure.
type Client[F Fault] struct {
	HTTP     *http.Client
	Token    *oauth2.Token
	Config   oauth2.Config
	newFault func() F
}

// New returns a Client[F] with a default http.Client and oauth2.Token.
func New[F Fault](newFault func() F) *Client[F] {
	return &Client[F]{
		HTTP:     &http.Client{},
		Token:    &oauth2.Token{},
		newFault: newFault,
	}
}

// Do executes the HTTP request and populates v with the decoded response body.
// If the response status indicates an error, a new F is decoded and returned.
func (c *Client[F]) Do(req *http.Request, v any) error {
	ctx := req.Context()
	res, err := c.HTTP.Do(req) //nolint:gosec // G704: false positive, taint analysis on http.Client
	if err != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return err
		}
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusBadRequest {
		f := c.newFault()
		if decErr := json.NewDecoder(res.Body).Decode(f); decErr != nil && !errors.Is(decErr, io.EOF) {
			return decErr
		}
		f.SetDefaults(res.StatusCode, http.StatusText(res.StatusCode))
		return f
	}

	if v != nil {
		if decErr := json.NewDecoder(res.Body).Decode(v); decErr != nil && !errors.Is(decErr, io.EOF) {
			return decErr
		}
	}
	return nil
}

// ApplyAutoRefresh replaces the HTTP client with one that automatically refreshes
// oauth2 tokens, using the Client's Config and Token.
func ApplyAutoRefresh[F Fault](ctx context.Context, c *Client[F]) error {
	c.HTTP = c.Config.Client(ctx, c.Token)
	return nil
}

// ApplyRateLimiter wraps the HTTP client's transport with a rate limiter.
func ApplyRateLimiter[F Fault](c *Client[F], r *rate.Limiter) error {
	if r == nil {
		return errors.New("nil limiter")
	}
	c.HTTP.Transport = &httpwares.RateLimitTransport{
		Limiter:   r,
		Transport: c.HTTP.Transport,
	}
	return nil
}

// ApplyHTTPTracing wraps the HTTP client's transport with verbose logging when debug is true.
func ApplyHTTPTracing[F Fault](c *Client[F], debug bool) error {
	if debug {
		c.HTTP.Transport = &httpwares.VerboseTransport{
			Transport: c.HTTP.Transport,
		}
	}
	return nil
}

// ApplyTransport sets the HTTP client's transport.
func ApplyTransport[F Fault](c *Client[F], t http.RoundTripper) error {
	if t == nil {
		return errors.New("nil transport")
	}
	c.HTTP.Transport = t
	return nil
}

// ApplyHTTPClient replaces the HTTP client entirely.
func ApplyHTTPClient[F Fault](c *Client[F], client *http.Client) error {
	if client == nil {
		return errors.New("nil client")
	}
	c.HTTP = client
	return nil
}
