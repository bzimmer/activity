package zwift

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/bzimmer/activity"
)

//go:generate genwith --do --client --token --ratelimit --config --endpoint-func --package zwift

const _baseURL = "https://us-or-rly101.zwift.com"
const userAgent = "CNL/3.4.1 (Darwin Kernel 20.3.0) zwift/1.0.61590 curl/7.64.1"

// Endpoint is Zwifts's OAuth 2.0 endpoint
func Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		TokenURL:  "https://secure.zwift.com/auth/realms/zwift/tokens/access/codes",
		AuthStyle: oauth2.AuthStyleAutoDetect,
	}
}

// Client for communicating with Zwift
type Client struct {
	token    *oauth2.Token
	client   *http.Client
	config   oauth2.Config
	baseURL  string
	username string
	password string

	lock sync.RWMutex

	Auth     *AuthService
	Activity *ActivityService
	Profile  *ProfileService
}

func (c *Client) Exporter() activity.Exporter {
	return c.Activity
}

func withServices() Option {
	return func(c *Client) error {
		c.Auth = &AuthService{c}
		c.Profile = &ProfileService{c}
		c.Activity = &ActivityService{c}
		c.token.TokenType = "bearer"
		if c.baseURL == "" {
			c.baseURL = _baseURL
		}
		c.lock = sync.RWMutex{}
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

// WithTokenRefresh refreshes the access token if none is provided
func WithTokenRefresh(username, password string) Option {
	return func(c *Client) error {
		c.username = username
		c.password = password
		return nil
	}
}

func (c *Client) validateToken(ctx context.Context) error {
	c.lock.RLock()
	// if no access token try to acquire one
	if c.token != nil && c.token.AccessToken != "" {
		c.lock.RUnlock()
		return nil
	}
	c.lock.RUnlock()
	// bail if username and password are not available
	if c.username == "" || c.password == "" {
		return errors.New("accessToken required")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	token, err := c.Auth.Refresh(ctx, c.username, c.password)
	if err != nil {
		return err
	}
	c.token = token
	return nil
}

func (c *Client) newAPIRequest(ctx context.Context, method, uri string) (*http.Request, error) {
	if err := c.validateToken(ctx); err != nil {
		return nil, err
	}
	u, err := url.Parse(fmt.Sprintf("%s/%s", c.baseURL, uri))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.token.AccessToken))
	return req, nil
}
