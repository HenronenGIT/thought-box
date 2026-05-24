package domain

import (
	"time"

	"github.com/google/uuid"
)

const SeededUserID = "00000000-0000-4000-8000-000000000001"

type Status string

const (
	StatusPending             Status = "pending"
	StatusTranscribing        Status = "transcribing"
	StatusEnriching           Status = "enriching"
	StatusDone                Status = "done"
	StatusFailedTranscription Status = "failed_transcription"
	StatusFailedEnrichment    Status = "failed_enrichment"
)

type Category string

const (
	CategoryIdea        Category = "idea"
	CategoryObservation Category = "observation"
	CategoryFeeling     Category = "feeling"
	CategoryLearning    Category = "learning"
)

func ParseCategory(value string) (Category, bool) {
	switch Category(value) {
	case CategoryIdea, CategoryObservation, CategoryFeeling, CategoryLearning:
		return Category(value), true
	default:
		return "", false
	}
}

type Thought struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	AudioS3Key    string
	MimeType      string
	DurationMs    *int64
	SizeBytes     int64
	Transcript    *string
	Status        Status
	Attempts      int
	LastError     *string
	TranscribedAt *time.Time
	Enrichment    *ThoughtEnrichment
}

type ThoughtEnrichment struct {
	Category      Category
	Tags          []string
	Title         string
	Summary       string
	Model         string
	PromptVersion string
}

type User struct {
	ID          uuid.UUID
	Email       string
	GoogleSub   string
	DisplayName string
	CreatedAt   time.Time
}

type AudioBlob struct {
	Key       string
	MimeType  string
	SizeBytes int64
}
