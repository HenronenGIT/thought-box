// Package google wraps the OAuth 2.0 dance with Google so the rest of the
// app sees a clean "give me an identity from this code" interface. The token
// and userinfo URLs are seams so handler tests can stand in a stub server
// without hitting Google.
package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	googleendpoint "golang.org/x/oauth2/google"
)

const defaultUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

type Identity struct {
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
}

type Client struct {
	cfg         *oauth2.Config
	userInfoURL string
	http        *http.Client
}

// New constructs a Client wired to Google's real endpoints.
func New(clientID, clientSecret, redirectURL string) *Client {
	return &Client{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     googleendpoint.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		},
		userInfoURL: defaultUserInfoURL,
		http:        http.DefaultClient,
	}
}

// NewForTest swaps Google's token + userinfo URLs and HTTP client for stubs.
func NewForTest(clientID, clientSecret, redirectURL, tokenURL, authURL, userInfoURL string, httpClient *http.Client) *Client {
	return &Client{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     oauth2.Endpoint{AuthURL: authURL, TokenURL: tokenURL},
			Scopes:       []string{"openid", "email", "profile"},
		},
		userInfoURL: userInfoURL,
		http:        httpClient,
	}
}

// LoginURL returns the URL the browser should be redirected to in order to
// start the OAuth dance. state is round-tripped back via the callback.
func (c *Client) LoginURL(state string) string {
	return c.cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange swaps a callback code for the authenticated user's identity.
func (c *Client) Exchange(ctx context.Context, code string) (Identity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.http)
	token, err := c.cfg.Exchange(ctx, code)
	if err != nil {
		return Identity{}, fmt.Errorf("token exchange: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
	if err != nil {
		return Identity{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return Identity{}, fmt.Errorf("userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Identity{}, fmt.Errorf("userinfo returned %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Identity{}, fmt.Errorf("userinfo decode: %w", err)
	}
	return Identity(payload), nil
}
