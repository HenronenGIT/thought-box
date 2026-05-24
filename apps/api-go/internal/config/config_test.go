package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

// validBase64Key32 is base64-encoded 32 zero bytes — a syntactically valid SESSION_SIGNING_KEY.
const validBase64Key32 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func baseEnv() []string {
	return []string{
		"DATABASE_URL=postgres://thoughts:thoughts@localhost:5432/thoughts_dev",
		"OPENAI_API_KEY=test",
		"S3_BUCKET=thoughts",
		"S3_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
		"GOOGLE_OAUTH_CLIENT_ID=client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET=client-secret",
		"GOOGLE_OAUTH_REDIRECT_URL=http://localhost:8080/auth/google/callback",
		"WEB_BASE_URL=http://localhost:3000",
		"SESSION_SIGNING_KEY=" + validBase64Key32,
	}
}

func TestFromMapRequiresPostgresURLWithCredentials(t *testing.T) {
	withoutDB := withoutEnv(baseEnv(), "DATABASE_URL")

	_, err := fromMap(append(withoutDB, "DATABASE_URL=jdbc:postgresql://localhost:5432/thoughts_dev"))
	if err == nil {
		t.Fatal("expected invalid URL")
	}

	cfg, err := fromMap(baseEnv())
	if err != nil {
		t.Fatalf("expected config, got %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port, got %s", cfg.Port)
	}
	if !strings.Contains(cfg.DatabaseURL, "default_query_exec_mode=simple_protocol") {
		t.Fatalf("expected simple protocol database URL, got %s", cfg.DatabaseURL)
	}
}

func TestFromMapPopulatesGoogleOAuthAndSessionFields(t *testing.T) {
	cfg, err := fromMap(baseEnv())
	if err != nil {
		t.Fatalf("expected config, got %v", err)
	}
	if cfg.GoogleOAuth.ClientID != "client-id" {
		t.Errorf("ClientID = %q", cfg.GoogleOAuth.ClientID)
	}
	if cfg.GoogleOAuth.ClientSecret != "client-secret" {
		t.Errorf("ClientSecret = %q", cfg.GoogleOAuth.ClientSecret)
	}
	if cfg.GoogleOAuth.RedirectURL != "http://localhost:8080/auth/google/callback" {
		t.Errorf("RedirectURL = %q", cfg.GoogleOAuth.RedirectURL)
	}
	if cfg.WebBaseURL != "http://localhost:3000" {
		t.Errorf("WebBaseURL = %q", cfg.WebBaseURL)
	}
	if len(cfg.SessionSigningKey) != 32 {
		t.Errorf("SessionSigningKey length = %d, want 32", len(cfg.SessionSigningKey))
	}
}

func TestFromMapRequiresAuthEnvVars(t *testing.T) {
	required := []string{
		"GOOGLE_OAUTH_CLIENT_ID",
		"GOOGLE_OAUTH_CLIENT_SECRET",
		"GOOGLE_OAUTH_REDIRECT_URL",
		"WEB_BASE_URL",
		"SESSION_SIGNING_KEY",
	}
	for _, name := range required {
		t.Run("missing_"+name, func(t *testing.T) {
			_, err := fromMap(withoutEnv(baseEnv(), name))
			if err == nil {
				t.Fatalf("expected error when %s missing", name)
			}
			if !strings.Contains(err.Error(), name) {
				t.Errorf("error should mention %s, got %v", name, err)
			}
		})
	}
}

func TestFromMapRejectsMalformedSessionSigningKey(t *testing.T) {
	cases := []struct {
		name string
		val  string
	}{
		{"not_base64", "***not-base64***"},
		{"wrong_length", base64Of(16)}, // 16 bytes, want 32
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := append(withoutEnv(baseEnv(), "SESSION_SIGNING_KEY"), "SESSION_SIGNING_KEY="+c.val)
			_, err := fromMap(env)
			if err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
			if !strings.Contains(err.Error(), "SESSION_SIGNING_KEY") {
				t.Errorf("error should mention SESSION_SIGNING_KEY, got %v", err)
			}
		})
	}
}

// base64Of returns base64-encoded n zero bytes.
func base64Of(n int) string {
	return base64.StdEncoding.EncodeToString(make([]byte, n))
}

// withoutEnv returns env with any pair whose key matches name removed.
func withoutEnv(env []string, name string) []string {
	out := make([]string, 0, len(env))
	for _, pair := range env {
		key, _, _ := strings.Cut(pair, "=")
		if key == name {
			continue
		}
		out = append(out, pair)
	}
	return out
}
