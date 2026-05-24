package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
)

func TestParseLimit(t *testing.T) {
	limit, err := parseLimit("")
	if err != nil || limit != 20 {
		t.Fatalf("default limit = %d, %v", limit, err)
	}
	if _, err := parseLimit("0"); err == nil {
		t.Fatal("expected invalid low limit")
	}
	if _, err := parseLimit("101"); err == nil {
		t.Fatal("expected invalid high limit")
	}
}

func TestHealthRouteAvoidsCloudRunReservedPath(t *testing.T) {
	router := NewRouter(Dependencies{Config: config.Config{AppEnv: "test"}, Logger: slog.Default()})

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("expected /health 200, got %d", health.Code)
	}

	healthz := httptest.NewRecorder()
	router.ServeHTTP(healthz, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthz.Code != http.StatusNotFound {
		t.Fatalf("expected /healthz unused locally, got %d", healthz.Code)
	}
}
