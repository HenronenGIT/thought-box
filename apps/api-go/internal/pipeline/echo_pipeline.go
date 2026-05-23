package pipeline

import (
	"context"
	"log/slog"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/echo"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/repository"
	"github.com/google/uuid"
)

type EchoPipeline struct {
	repo                 *repository.EchoRepository
	generator            echo.Generator
	logger               *slog.Logger
	triggerCategories    []string
	generatorModel       string
	generatorPromptVer   string
}

func NewEchoPipeline(repo *repository.EchoRepository, generator echo.Generator, logger *slog.Logger, triggerCategories []domain.Category, model string, promptVersion string) EchoPipeline {
	cats := make([]string, 0, len(triggerCategories))
	for _, c := range triggerCategories {
		cats = append(cats, string(c))
	}
	return EchoPipeline{
		repo:                 repo,
		generator:            generator,
		logger:               logger,
		triggerCategories:    cats,
		generatorModel:       model,
		generatorPromptVer:   promptVersion,
	}
}

func (p EchoPipeline) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		didWork, err := p.processOne(ctx)
		if err != nil {
			p.logger.Error("echo pipeline failed", "error", err)
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

func (p EchoPipeline) processOne(ctx context.Context) (bool, error) {
	didWork, err := p.seedDefault(ctx)
	if didWork || err != nil {
		return didWork, err
	}
	return p.generateOne(ctx)
}

func (p EchoPipeline) seedDefault(ctx context.Context) (bool, error) {
	thought, err := p.repo.NextThoughtNeedingDefault(ctx, p.triggerCategories)
	if err != nil || thought == nil {
		return false, err
	}
	if thought.Enrichment == nil {
		return true, nil
	}
	mode, ok := domain.DefaultModeFor(thought.Enrichment.Category)
	if !ok {
		return true, nil
	}
	if err := p.repo.InsertDefault(ctx, uuid.New(), thought.ID, mode); err != nil {
		return true, err
	}
	p.logger.Info("echo_seeded", "thought_id", thought.ID, "mode", mode)
	return true, nil
}

func (p EchoPipeline) generateOne(ctx context.Context) (bool, error) {
	job, err := p.repo.NextPending(ctx)
	if err != nil || job == nil {
		return false, err
	}
	content, err := p.generator.Generate(ctx, job.Transcript, job.Echo.Mode)
	if err != nil {
		return true, p.handleFailure(ctx, job.Echo, err)
	}
	if err := p.repo.MarkReady(ctx, job.Echo.ID, content, p.generatorModel, p.generatorPromptVer); err != nil {
		return true, err
	}
	p.logger.Info("echo_ready", "echo_id", job.Echo.ID, "thought_id", job.Echo.ThoughtID, "mode", job.Echo.Mode)
	return true, nil
}

func (p EchoPipeline) handleFailure(ctx context.Context, e domain.Echo, cause error) error {
	attempts := e.Attempts + 1
	decision := retryDecision(e.Attempts)
	if !decision.Retry {
		p.logger.Warn("echo_terminal_failure", "echo_id", e.ID, "thought_id", e.ThoughtID, "attempts", attempts, "error", cause)
		return p.repo.RecordFailure(ctx, e.ID, attempts, cause.Error())
	}
	if err := p.repo.RecordRetry(ctx, e.ID, attempts, cause.Error()); err != nil {
		return err
	}
	p.logger.Warn("echo_retry", "echo_id", e.ID, "thought_id", e.ThoughtID, "attempts", attempts, "error", cause)
	select {
	case <-ctx.Done():
	case <-time.After(decision.Delay):
	}
	return nil
}
