package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/auth/google"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/auth/state"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/google/uuid"
)

func TestProtectedRouteWithoutCookieReturns401(t *testing.T) {
	router := NewRouter(Dependencies{
		Config:   config.Config{AppEnv: "test"},
		Logger:   slog.Default(),
		Sessions: &staticSessions{UserID: uuid.New()},
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestProtectedRouteWithStubCookieReturns200(t *testing.T) {
	userID := uuid.New()
	router := NewRouter(Dependencies{
		Config:   config.Config{AppEnv: "test"},
		Logger:   slog.Default(),
		Sessions: &staticSessions{UserID: userID},
		Users:    &recordingUsers{},
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "stub"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["user_id"] != userID.String() {
		t.Errorf("user_id = %q", body["user_id"])
	}
}

func TestGoogleLoginRedirectsAndSetsStateCookie(t *testing.T) {
	deps := authTestDeps(t, &recordingAllowlist{allowed: true})
	router := NewRouter(deps)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/google/login", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/authorize") || !strings.Contains(location, "client_id=test-client-id") {
		t.Errorf("Location should point at stub Google's authorize endpoint: %q", location)
	}
	if findCookie(rec.Result().Cookies(), "oauth_state") == nil {
		t.Error("oauth_state cookie not set")
	}
}

func TestGoogleCallbackHappyPathIssuesSession(t *testing.T) {
	allowlist := &recordingAllowlist{allowed: true}
	deps := authTestDeps(t, allowlist)
	router := NewRouter(deps)

	stateToken, nonce := freshStateToken(t, deps)
	req := httptest.NewRequest(http.MethodGet,
		"/auth/google/callback?code=code-ok&state="+nonce, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: stateToken})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "https://web.test/" && loc != "https://web.test" {
		t.Errorf("Location = %q, want web base URL", loc)
	}
	if findCookie(rec.Result().Cookies(), "session") == nil {
		t.Error("session cookie was not set on happy path")
	}
	users := deps.Users.(*recordingUsers)
	if users.lastEmail != "user@example.com" {
		t.Errorf("user upsert email = %q", users.lastEmail)
	}
}

func TestGoogleCallbackRejectsStateMismatch(t *testing.T) {
	deps := authTestDeps(t, &recordingAllowlist{allowed: true})
	router := NewRouter(deps)

	stateToken, _ := freshStateToken(t, deps)
	req := httptest.NewRequest(http.MethodGet,
		"/auth/google/callback?code=code-ok&state=wrong-nonce", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: stateToken})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Location"), "error=state_mismatch") {
		t.Errorf("Location = %q", rec.Header().Get("Location"))
	}
	if findCookie(rec.Result().Cookies(), "session") != nil {
		t.Error("session cookie set despite state mismatch")
	}
}

func TestGoogleCallbackRejectsNotAllowlistedEmail(t *testing.T) {
	deps := authTestDeps(t, &recordingAllowlist{allowed: false})
	router := NewRouter(deps)

	stateToken, nonce := freshStateToken(t, deps)
	req := httptest.NewRequest(http.MethodGet,
		"/auth/google/callback?code=code-ok&state="+nonce, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: stateToken})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Location"), "error=not_allowed") {
		t.Errorf("Location = %q", rec.Header().Get("Location"))
	}
	if findCookie(rec.Result().Cookies(), "session") != nil {
		t.Error("session cookie set despite not-allowed email")
	}
}

func TestLogoutClearsCookieAndRevokesSession(t *testing.T) {
	sessions := &recordingSessions{userID: uuid.New()}
	router := NewRouter(Dependencies{
		Config:   config.Config{AppEnv: "test"},
		Logger:   slog.Default(),
		Sessions: sessions,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "token-to-revoke"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	c := findCookie(rec.Result().Cookies(), "session")
	if c == nil || c.MaxAge >= 0 {
		t.Errorf("session cookie should be cleared, got %+v", c)
	}
	if sessions.revoked != "token-to-revoke" {
		t.Errorf("expected Revoke called with raw token, got %q", sessions.revoked)
	}
}

// --- shared test helpers ---

func authTestDeps(t *testing.T, gate allowlistGate) Dependencies {
	t.Helper()
	stub := newStubGoogle(t)
	t.Cleanup(stub.Close)

	signingKey := []byte("test-key-test-key-test-key-test!")
	clientID := "test-client-id"
	clientSecret := "test-client-secret"
	redirectURL := "https://api.test/auth/google/callback"

	return Dependencies{
		Config: config.Config{
			AppEnv:     "test",
			WebBaseURL: "https://web.test",
		},
		Logger:    slog.Default(),
		Sessions:  &recordingSessions{userID: uuid.New()},
		Users:     &recordingUsers{},
		Allowlist: gate,
		State:     state.New(signingKey, nil),
		Google: google.NewForTest(
			clientID, clientSecret, redirectURL,
			stub.URL+"/token",
			stub.URL+"/authorize",
			stub.URL+"/userinfo",
			http.DefaultClient,
		),
	}
}

func freshStateToken(t *testing.T, deps Dependencies) (token, nonce string) {
	t.Helper()
	nonce = "fixed-nonce-abc"
	return deps.State.Sign(nonce, 5*time.Minute), nonce
}

// newStubGoogle stands in for Google's token + userinfo endpoints.
func newStubGoogle(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = r.ParseForm()
			if r.PostForm.Get("code") != "code-ok" {
				http.Error(w, "bad code", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"at-123","token_type":"Bearer","expires_in":3600}`))
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer at-123" {
				http.Error(w, "bad bearer", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"sub":"google-sub-xyz",
				"email":"user@example.com",
				"email_verified":true,
				"name":"Test User"
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// --- recording fakes ---

type recordingUsers struct {
	mu        sync.Mutex
	lastSub   string
	lastEmail string
	lastName  string
}

func (r *recordingUsers) UpsertByGoogleSub(ctx context.Context, sub, email, name string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSub, r.lastEmail, r.lastName = sub, email, name
	return domain.User{ID: uuid.New(), Email: email, GoogleSub: sub, DisplayName: name}, nil
}

func (r *recordingUsers) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return &domain.User{ID: id}, nil
}

type recordingAllowlist struct{ allowed bool }

func (r *recordingAllowlist) IsAllowed(ctx context.Context, email string) (bool, error) {
	return r.allowed, nil
}

type recordingSessions struct {
	userID  uuid.UUID
	issued  string
	revoked string
}

func (r *recordingSessions) Issue(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	r.issued = "session-" + userID.String()
	return r.issued, nil
}
func (r *recordingSessions) Lookup(ctx context.Context, raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, errStubSession
	}
	return r.userID, nil
}
func (r *recordingSessions) Revoke(ctx context.Context, raw string) error {
	r.revoked = raw
	return nil
}

// silence unused-import warning for url.Parse in some refactors
var _ = url.Parse
