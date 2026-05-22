package httpapi

import (
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
)

type thoughtListResponse struct {
	Items      []thoughtResponse `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type thoughtResponse struct {
	ID         string              `json:"id"`
	Status     string              `json:"status"`
	CreatedAt  string              `json:"created_at"`
	UpdatedAt  string              `json:"updated_at"`
	Transcript *string             `json:"transcript"`
	Audio      audioResponse       `json:"audio"`
	Enrichment *enrichmentResponse `json:"enrichment"`
	LastError  *string             `json:"last_error"`
}

type audioResponse struct {
	MimeType   string `json:"mime_type"`
	DurationMs *int64 `json:"duration_ms"`
	SizeBytes  int64  `json:"size_bytes"`
}

type enrichmentResponse struct {
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Model         string   `json:"model"`
	PromptVersion string   `json:"prompt_version"`
}

func newThoughtResponse(thought domain.Thought) thoughtResponse {
	response := thoughtResponse{
		ID:         thought.ID.String(),
		Status:     string(thought.Status),
		CreatedAt:  thought.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:  thought.UpdatedAt.Format(time.RFC3339Nano),
		Transcript: thought.Transcript,
		Audio: audioResponse{
			MimeType:   thought.MimeType,
			DurationMs: thought.DurationMs,
			SizeBytes:  thought.SizeBytes,
		},
		LastError: thought.LastError,
	}
	if thought.Enrichment != nil {
		response.Enrichment = &enrichmentResponse{
			Category:      string(thought.Enrichment.Category),
			Tags:          thought.Enrichment.Tags,
			Title:         thought.Enrichment.Title,
			Summary:       thought.Enrichment.Summary,
			Model:         thought.Enrichment.Model,
			PromptVersion: thought.Enrichment.PromptVersion,
		}
	}
	return response
}

func writeJSON(w httpResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = jsonEncoder(w).Encode(payload)
}

func writeError(w httpResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
