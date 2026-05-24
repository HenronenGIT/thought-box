package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/user"
)

const (
	correlationIDHeader = "X-Correlation-Id"
	sessionCookieName   = "session"
	stateCookieName     = "oauth_state"
)

func (s Server) correlationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get(correlationIDHeader)
		if correlationID == "" {
			correlationID = randomID()
		}
		w.Header().Set(correlationIDHeader, correlationID)
		next.ServeHTTP(w, r)
	})
}

func (s Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered", "panic", recovered)
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s Server) cors(next http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, origin := range s.config.CorsAllowedOrigins {
		if parsed, err := url.Parse(origin); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			allowed[origin] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Correlation-Id")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			if origin != "" && !slices.Contains(s.config.CorsAllowedOrigins, origin) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireSession reads the session cookie, validates it, and attaches the
// resolved user id to the request context. Any failure clears the cookie and
// returns 401 so the client can re-authenticate.
func (s Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		userID, err := s.sessions.Lookup(r.Context(), cookie.Value)
		if err != nil {
			clearSessionCookie(w, s.config)
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		ctx := user.WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func setSessionCookie(w http.ResponseWriter, cfg config.Config, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd(cfg),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd(cfg),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func setStateCookie(w http.ResponseWriter, cfg config.Config, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    value,
		Path:     "/auth/google",
		HttpOnly: true,
		Secure:   isProd(cfg),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearStateCookie(w http.ResponseWriter, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/auth/google",
		HttpOnly: true,
		Secure:   isProd(cfg),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func isProd(cfg config.Config) bool {
	return cfg.AppEnv == "prod" || cfg.AppEnv == "production"
}

func randomID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(bytes[:])
}

