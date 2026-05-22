package transcription

import (
	"context"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
)

type Result struct {
	Text  string
	Model string
}

type Transcriber interface {
	Transcribe(ctx context.Context, audio domain.AudioBlob) (Result, error)
}
