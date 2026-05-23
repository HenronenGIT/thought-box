package httpapi

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/repository"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/user"
	"github.com/google/uuid"
)

type fakeEchoes struct {
	requestErr error
	created    *domain.Echo
}

func (f *fakeEchoes) ListByThought(ctx context.Context, userID, thoughtID uuid.UUID) ([]domain.Echo, error) {
	return nil, nil
}

func (f *fakeEchoes) RequestEcho(ctx context.Context, userID, thoughtID uuid.UUID, mode domain.EchoMode) (*domain.Echo, error) {
	if f.requestErr != nil {
		return nil, f.requestErr
	}
	echo := domain.Echo{
		ID:        uuid.New(),
		ThoughtID: thoughtID,
		Mode:      mode,
		Status:    domain.EchoStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	f.created = &echo
	return &echo, nil
}

func newTestServer(echoes echoesStore) http.Handler {
	return NewRouter(config.Config{AppEnv: "test"}, slog.Default(), nil, echoes, nil, user.SeededResolver{})
}

func TestCreateEchoInvalidMode(t *testing.T) {
	router := newTestServer(&fakeEchoes{})
	req := httptest.NewRequest(http.MethodPost, "/thoughts/"+uuid.New().String()+"/echoes", bytes.NewBufferString(`{"mode":"bogus"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEchoDuplicateConflict(t *testing.T) {
	router := newTestServer(&fakeEchoes{requestErr: repository.ErrEchoDuplicate})
	req := httptest.NewRequest(http.MethodPost, "/thoughts/"+uuid.New().String()+"/echoes", bytes.NewBufferString(`{"mode":"mirror"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEchoCapConflict(t *testing.T) {
	router := newTestServer(&fakeEchoes{requestErr: repository.ErrEchoCapReached})
	req := httptest.NewRequest(http.MethodPost, "/thoughts/"+uuid.New().String()+"/echoes", bytes.NewBufferString(`{"mode":"challenger"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestCreateEchoSuccess(t *testing.T) {
	store := &fakeEchoes{}
	router := newTestServer(store)
	req := httptest.NewRequest(http.MethodPost, "/thoughts/"+uuid.New().String()+"/echoes", bytes.NewBufferString(`{"mode":"reframer"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.created == nil || store.created.Mode != domain.EchoModeReframer {
		t.Fatal("expected reframer echo to be created")
	}
}
