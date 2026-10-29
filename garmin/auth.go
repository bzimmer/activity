package garmin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/bzimmer/activity"
)

const tokenURI = "personalapitoken-service/personal-api/personal-api-key/token"

// AuthService is the API for auth endpoints
type AuthService service

// Token exchanges the API key for a bearer token, which the client then
// uses for every request until it expires.
func (s *AuthService) Token(ctx context.Context) (*oauth2.Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.client.apiURL+"/"+tokenURI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", activity.UserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Authorization", "ApiKey "+s.client.apiKey)
	res, err := s.client.send(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	jwt := strings.TrimSpace(string(body))
	if jwt == "" {
		return nil, errors.New("the token endpoint returned an empty token")
	}
	token := &oauth2.Token{AccessToken: jwt, TokenType: "Bearer", Expiry: expiry(jwt)}
	s.client.base.Token = token
	return token, nil
}

// expiry reads the exp claim of a JWT, zero (never expiring) when it has
// none or doesn't parse; the signature is Garmin's to check, not ours.
func expiry(jwt string) time.Time {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}
