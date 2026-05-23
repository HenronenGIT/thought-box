package domain

import (
	"time"

	"github.com/google/uuid"
)

type EchoMode string

const (
	EchoModeMirror     EchoMode = "mirror"
	EchoModeChallenger EchoMode = "challenger"
	EchoModeReframer   EchoMode = "reframer"
	EchoModeExtender   EchoMode = "extender"
)

func ParseEchoMode(value string) (EchoMode, bool) {
	switch EchoMode(value) {
	case EchoModeMirror, EchoModeChallenger, EchoModeReframer, EchoModeExtender:
		return EchoMode(value), true
	default:
		return "", false
	}
}

type EchoStatus string

const (
	EchoStatusPending    EchoStatus = "pending"
	EchoStatusGenerating EchoStatus = "generating"
	EchoStatusReady      EchoStatus = "ready"
	EchoStatusFailed     EchoStatus = "failed"
)

const MaxEchoesPerThought = 4

func DefaultModeFor(category Category) (EchoMode, bool) {
	switch category {
	case CategoryFeeling:
		return EchoModeMirror, true
	case CategoryIdea:
		return EchoModeChallenger, true
	case CategoryObservation:
		return EchoModeReframer, true
	case CategoryLearning:
		return EchoModeExtender, true
	default:
		return "", false
	}
}

type Echo struct {
	ID            uuid.UUID
	ThoughtID     uuid.UUID
	Mode          EchoMode
	Content       *string
	Status        EchoStatus
	IsDefault     bool
	Attempts      int
	LastError     *string
	Model         *string
	PromptVersion *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
