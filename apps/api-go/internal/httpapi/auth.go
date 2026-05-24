package httpapi

import (
	"net/http"
	"net/url"
	"time"
)

const (
	stateCookieTTL = 5 * time.Minute
	sessionTTL     = 30 * 24 * time.Hour
)

func (s Server) googleLogin(w http.ResponseWriter, r *http.Request) {
	if s.google == nil || s.state == nil {
		writeError(w, http.StatusInternalServerError, "Auth not configured")
		return
	}
	nonce := randomID()
	token := s.state.Sign(nonce, stateCookieTTL)
	setStateCookie(w, s.config, token, stateCookieTTL)
	http.Redirect(w, r, s.google.LoginURL(nonce), http.StatusFound)
}

func (s Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	if s.google == nil || s.state == nil || s.sessions == nil || s.users == nil || s.allowlist == nil {
		writeError(w, http.StatusInternalServerError, "Auth not configured")
		return
	}

	cookie, err := r.Cookie(stateCookieName)
	if err != nil {
		s.redirectLogin(w, r, "state_missing")
		return
	}
	clearStateCookie(w, s.config)

	nonce, err := s.state.Verify(cookie.Value)
	if err != nil {
		s.redirectLogin(w, r, "state_invalid")
		return
	}
	if r.URL.Query().Get("state") != nonce {
		s.redirectLogin(w, r, "state_mismatch")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		s.redirectLogin(w, r, "missing_code")
		return
	}

	identity, err := s.google.Exchange(r.Context(), code)
	if err != nil {
		s.logger.Error("google exchange failed", "error", err)
		s.redirectLogin(w, r, "exchange_failed")
		return
	}
	if !identity.EmailVerified {
		s.redirectLogin(w, r, "email_unverified")
		return
	}

	allowed, err := s.allowlist.IsAllowed(r.Context(), identity.Email)
	if err != nil {
		s.logger.Error("allowlist check failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if !allowed {
		s.redirectLogin(w, r, "not_allowed")
		return
	}

	u, err := s.users.UpsertByGoogleSub(r.Context(), identity.Sub, identity.Email, identity.Name)
	if err != nil {
		s.logger.Error("user upsert failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	token, err := s.sessions.Issue(r.Context(), u.ID, sessionTTL)
	if err != nil {
		s.logger.Error("session issue failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	setSessionCookie(w, s.config, token, sessionTTL)
	http.Redirect(w, r, s.config.WebBaseURL, http.StatusFound)
}

func (s Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" && s.sessions != nil {
		_ = s.sessions.Revoke(r.Context(), cookie.Value)
	}
	clearSessionCookie(w, s.config)
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) redirectLogin(w http.ResponseWriter, r *http.Request, reason string) {
	base := s.config.WebBaseURL
	if base == "" {
		writeError(w, http.StatusBadRequest, reason)
		return
	}
	u, err := url.Parse(base)
	if err != nil {
		writeError(w, http.StatusBadRequest, reason)
		return
	}
	u.Path = "/login"
	q := u.Query()
	q.Set("error", reason)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}
