package transcription

import (
	"context"
	"fmt"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/storage"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAITranscriber struct {
	client openai.Client
	store  storage.Store
}

func NewOpenAITranscriber(apiKey string, store storage.Store) OpenAITranscriber {
	return OpenAITranscriber{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		store:  store,
	}
}

func (t OpenAITranscriber) Transcribe(ctx context.Context, audio domain.AudioBlob) (Result, error) {
	stored, err := t.store.Get(ctx, audio.Key)
	if err != nil {
		return Result{}, err
	}
	defer stored.Bytes.Close()

	result, err := t.client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:  openai.File(stored.Bytes, fmt.Sprintf("thought.%s", storage.ExtensionForMimeType(audio.MimeType)), audio.MimeType),
		Model: openai.AudioModelWhisper1,
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Text: result.Text, Model: string(openai.AudioModelWhisper1)}, nil
}
