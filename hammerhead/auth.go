package hammerhead

import (
	"context"

	"golang.org/x/oauth2"
)

// AuthService is the API for auth endpoints
type AuthService service

// Refresh returns a new access token
func (s *AuthService) Refresh(ctx context.Context) (*oauth2.Token, error) {
	t := s.client.base.Config.TokenSource(ctx, s.client.base.Token)
	t = oauth2.ReuseTokenSource(s.client.base.Token, t)
	return t.Token()
}
