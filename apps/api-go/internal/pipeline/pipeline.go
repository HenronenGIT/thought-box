package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/enrichment"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/repository"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/transcription"
)

type Pipeline struct {
	repo        *repository.ThoughtRepository
	transcriber transcription.Transcriber
	enricher    enrichment.Enricher
	logger      *slog.Logger
}

func New(repo *repository.ThoughtRepository, transcriber transcription.Transcriber, enricher enrichment.Enricher, logger *slog.Logger) Pipeline {
	return Pipeline{repo: repo, transcriber: transcriber, enricher: enricher, logger: logger}
}

func (p Pipeline) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		didWork, err := p.processOne(ctx)
		if err != nil {
			p.logger.Error("pipeline failed", "error", err)
		}
		if didWork {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p Pipeline) processOne(ctx context.Context) (bool, error) {
	didWork, err := p.processOneTranscription(ctx)
	if didWork || err != nil {
		return didWork, err
	}
	return p.processOneEnrichment(ctx)
}

func (p Pipeline) processOneTranscription(ctx context.Context) (bool, error) {
	thought, err := p.repo.NextPending(ctx)
	if err != nil || thought == nil {
		return false, err
	}
	result, err := p.transcriber.Transcribe(ctx, domain.AudioBlob{
		Key:       thought.AudioS3Key,
		MimeType:  thought.MimeType,
		SizeBytes: thought.SizeBytes,
	})
	if err != nil {
		return true, p.handleFailure(ctx, *thought, domain.StatusFailedTranscription, err)
	}
	if err := p.repo.MarkTranscribed(ctx, thought.ID, result.Text); err != nil {
		return true, err
	}
	p.logger.Info("pipeline_transition", "thought_id", thought.ID, "from_status", "transcribing", "to_status", "enriching")
	return true, nil
}

func (p Pipeline) processOneEnrichment(ctx context.Context) (bool, error) {
	thought, err := p.repo.NextEnriching(ctx)
	if err != nil || thought == nil {
		return false, err
	}
	if thought.Transcript == nil {
		return true, p.handleFailure(ctx, *thought, domain.StatusFailedEnrichment, errors.New("missing transcript"))
	}
	result, err := p.enricher.Enrich(ctx, thought.ID, *thought.Transcript)
	if err != nil {
		return true, p.handleFailure(ctx, *thought, domain.StatusFailedEnrichment, err)
	}
	if err := p.repo.MarkEnriched(ctx, thought.ID, result); err != nil {
		return true, err
	}
	p.logger.Info("pipeline_transition", "thought_id", thought.ID, "from_status", "enriching", "to_status", "done")
	return true, nil
}

func (p Pipeline) handleFailure(ctx context.Context, thought domain.Thought, terminalStatus domain.Status, cause error) error {
	attempts := thought.Attempts + 1
	decision := retryDecision(thought.Attempts)
	if !decision.Retry {
		p.logger.Warn("pipeline_terminal_failure", "thought_id", thought.ID, "status", terminalStatus, "attempts", attempts, "error", cause)
		return p.repo.RecordFailure(ctx, thought.ID, terminalStatus, attempts, cause.Error())
	}
	retryStatus := domain.StatusEnriching
	if terminalStatus == domain.StatusFailedTranscription {
		retryStatus = domain.StatusPending
	}
	if err := p.repo.RecordRetry(ctx, thought.ID, retryStatus, attempts, cause.Error()); err != nil {
		return err
	}
	p.logger.Warn("pipeline_retry", "thought_id", thought.ID, "attempts", attempts, "error", cause)
	select {
	case <-ctx.Done():
	case <-time.After(decision.Delay):
	}
	return nil
}
