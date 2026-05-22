package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/repository"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/storage"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/user"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/validation"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Server struct {
	config       config.Config
	logger       *slog.Logger
	repository   *repository.ThoughtRepository
	blobStore    storage.Store
	userResolver user.Resolver
}

func NewRouter(cfg config.Config, logger *slog.Logger, repo *repository.ThoughtRepository, store storage.Store, resolver user.Resolver) http.Handler {
	server := Server{config: cfg, logger: logger, repository: repo, blobStore: store, userResolver: resolver}
	router := chi.NewRouter()
	router.Use(server.recoverPanic)
	router.Use(server.correlationID)
	router.Use(server.cors)
	router.Get("/health", server.health)
	router.Get("/config", server.clientConfig)
	router.Get("/me", server.me)
	router.Post("/thoughts", server.createThought)
	router.Get("/thoughts", server.listThoughts)
	router.Get("/thoughts/{id}", server.getThought)
	router.Get("/thoughts/{id}/audio", server.getThoughtAudio)
	return router
}

func (s Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "env": s.config.AppEnv})
}

func (s Server) clientConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.config.Limits)
}

func (s Server) me(w http.ResponseWriter, r *http.Request) {
	userID, err := s.userResolver.CurrentUserID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": userID.String()})
}

func (s Server) createThought(w http.ResponseWriter, r *http.Request) {
	userID, err := s.userResolver.CurrentUserID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if err := r.ParseMultipartForm(s.config.Limits.MaxSizeBytes + 1024*1024); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid multipart request")
		return
	}

	mimeType := r.FormValue("mime_type")
	durationMs, err := strconv.ParseInt(r.FormValue("duration_ms"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "duration_ms is required")
		return
	}
	if err := validation.ValidateMimeType(mimeType); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateDuration(durationMs, s.config.Limits); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		writeError(w, http.StatusBadRequest, "audio is required")
		return
	}
	defer file.Close()

	temp, sizeBytes, err := writeTempAudio(file)
	if err != nil {
		s.logger.Error("temp audio failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	if err := validation.ValidateSize(sizeBytes, s.config.Limits); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := temp.Seek(0, 0); err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	id := uuid.New()
	key := s.config.AppEnv + "/" + userID.String() + "/" + id.String()
	if err := s.blobStore.Put(r.Context(), key, mimeType, sizeBytes, temp); err != nil {
		s.logger.Error("audio upload failed", "error", err, "filename", header.Filename)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	thought, err := s.repository.InsertThought(r.Context(), id, userID, key, mimeType, durationMs, sizeBytes)
	if err != nil {
		s.logger.Error("insert thought failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":         thought.ID.String(),
		"status":     string(thought.Status),
		"created_at": thought.CreatedAt.Format(time.RFC3339Nano),
	})
}

func (s Server) listThoughts(w http.ResponseWriter, r *http.Request) {
	userID, err := s.userResolver.CurrentUserID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var before *time.Time
	if raw := r.URL.Query().Get("before"); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid before cursor")
			return
		}
		before = &parsed
	}

	var category *domain.Category
	if raw := r.URL.Query().Get("category"); raw != "" {
		parsed, ok := domain.ParseCategory(raw)
		if !ok {
			writeError(w, http.StatusBadRequest, "Invalid category")
			return
		}
		category = &parsed
	}

	thoughts, err := s.repository.ListThoughts(r.Context(), userID, limit+1, before, category, r.URL.Query().Get("tag"))
	if err != nil {
		s.logger.Error("list thoughts failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	nextCursor := ""
	if len(thoughts) > limit {
		nextCursor = thoughts[limit-1].CreatedAt.Format(time.RFC3339Nano)
		thoughts = thoughts[:limit]
	}
	response := thoughtListResponse{Items: make([]thoughtResponse, 0, len(thoughts))}
	if nextCursor != "" {
		response.NextCursor = &nextCursor
	}
	for _, thought := range thoughts {
		response.Items = append(response.Items, newThoughtResponse(thought))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s Server) getThought(w http.ResponseWriter, r *http.Request) {
	userID, err := s.userResolver.CurrentUserID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid thought id")
		return
	}

	thought, err := s.repository.FindThought(r.Context(), userID, id)
	if err != nil {
		s.logger.Error("find thought failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if thought == nil {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	writeJSON(w, http.StatusOK, newThoughtResponse(*thought))
}

func (s Server) getThoughtAudio(w http.ResponseWriter, r *http.Request) {
	userID, err := s.userResolver.CurrentUserID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid thought id")
		return
	}
	thought, err := s.repository.FindThought(r.Context(), userID, id)
	if err != nil {
		s.logger.Error("find thought audio failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if thought == nil {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	audio, err := s.blobStore.Get(r.Context(), thought.AudioS3Key)
	if err != nil {
		s.logger.Error("get audio failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer audio.Bytes.Close()
	w.Header().Set("Content-Type", audio.ContentType)
	if audio.ContentLength != nil {
		w.Header().Set("Content-Length", strconv.FormatInt(*audio.ContentLength, 10))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, audio.Bytes)
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return 20, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > 100 {
		return 0, errors.New("Invalid limit")
	}
	return limit, nil
}

func writeTempAudio(file multipart.File) (*os.File, int64, error) {
	temp, err := os.CreateTemp("", "thought-*.audio")
	if err != nil {
		return nil, 0, err
	}
	sizeBytes, err := io.Copy(temp, file)
	if err != nil {
		temp.Close()
		os.Remove(temp.Name())
		return nil, 0, err
	}
	return temp, sizeBytes, nil
}
