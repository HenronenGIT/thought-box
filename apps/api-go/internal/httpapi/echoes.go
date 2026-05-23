package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type echoesStore interface {
	ListByThought(ctx context.Context, userID uuid.UUID, thoughtID uuid.UUID) ([]domain.Echo, error)
	RequestEcho(ctx context.Context, userID uuid.UUID, thoughtID uuid.UUID, mode domain.EchoMode) (*domain.Echo, error)
}

type echoListResponse struct {
	Items []echoResponse `json:"items"`
}

type echoResponse struct {
	ID        string  `json:"id"`
	ThoughtID string  `json:"thought_id"`
	Mode      string  `json:"mode"`
	Content   *string `json:"content"`
	Status    string  `json:"status"`
	IsDefault bool    `json:"is_default"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func newEchoResponse(e domain.Echo) echoResponse {
	return echoResponse{
		ID:        e.ID.String(),
		ThoughtID: e.ThoughtID.String(),
		Mode:      string(e.Mode),
		Content:   e.Content,
		Status:    string(e.Status),
		IsDefault: e.IsDefault,
		CreatedAt: e.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: e.UpdatedAt.Format(time.RFC3339Nano),
	}
}

type createEchoRequest struct {
	Mode string `json:"mode"`
}

func (s Server) createEcho(w http.ResponseWriter, r *http.Request) {
	userID, err := s.userResolver.CurrentUserID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	thoughtID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid thought id")
		return
	}
	var body createEchoRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	mode, ok := domain.ParseEchoMode(body.Mode)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid mode")
		return
	}

	echo, err := s.echoes.RequestEcho(r.Context(), userID, thoughtID, mode)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEchoDuplicate):
			writeError(w, http.StatusConflict, "Echo of that mode already exists")
		case errors.Is(err, repository.ErrEchoCapReached):
			writeError(w, http.StatusConflict, "Maximum echoes reached for this thought")
		case errors.Is(err, repository.ErrThoughtNotReady):
			writeError(w, http.StatusConflict, "Thought is not ready for echoes")
		default:
			s.logger.Error("create echo failed", "error", err)
			writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, newEchoResponse(*echo))
}

func (s Server) listEchoes(w http.ResponseWriter, r *http.Request) {
	userID, err := s.userResolver.CurrentUserID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	thoughtID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid thought id")
		return
	}
	thought, err := s.repository.FindThought(r.Context(), userID, thoughtID)
	if err != nil {
		s.logger.Error("find thought failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if thought == nil {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	echoes, err := s.echoes.ListByThought(r.Context(), userID, thoughtID)
	if err != nil {
		s.logger.Error("list echoes failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	response := echoListResponse{Items: make([]echoResponse, 0, len(echoes))}
	for _, e := range echoes {
		response.Items = append(response.Items, newEchoResponse(e))
	}
	writeJSON(w, http.StatusOK, response)
}
