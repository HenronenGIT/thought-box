package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ThoughtRepository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *ThoughtRepository {
	return &ThoughtRepository{pool: pool}
}

func (r *ThoughtRepository) InsertThought(ctx context.Context, id uuid.UUID, userID uuid.UUID, audioKey string, mimeType string, durationMs int64, sizeBytes int64) (*domain.Thought, error) {
	rows, err := r.pool.Query(ctx, `
insert into thoughts (id, user_id, audio_s3_key, mime_type, duration_ms, size_bytes, status)
values ($1, $2, $3, $4, $5, $6, 'pending')
returning id, user_id, created_at, updated_at, audio_s3_key, mime_type,
    duration_ms, size_bytes, transcript, status, attempts, last_error,
    transcribed_at, null::text as category, null::text[] as tags, null::text as title,
    null::text as summary, null::text as model, null::text as prompt_version`,
		id, userID, audioKey, mimeType, durationMs, sizeBytes,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	thought, err := pgx.CollectOneRow(rows, scanThought)
	if err != nil {
		return nil, err
	}
	return &thought, nil
}

func (r *ThoughtRepository) FindThought(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*domain.Thought, error) {
	rows, err := r.pool.Query(ctx, selectThoughtSQL()+" where t.user_id = $1 and t.id = $2", userID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	thought, err := pgx.CollectOneRow(rows, scanThought)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &thought, nil
}

func (r *ThoughtRepository) ListThoughts(ctx context.Context, userID uuid.UUID, limit int, before *time.Time, category *domain.Category, tag string) ([]domain.Thought, error) {
	args := []any{userID}
	filters := []string{"t.user_id = $1"}
	nextArg := 2

	if before != nil {
		filters = append(filters, fmt.Sprintf("t.created_at < $%d", nextArg))
		args = append(args, *before)
		nextArg++
	}
	if category != nil {
		filters = append(filters, fmt.Sprintf("e.category = $%d", nextArg))
		args = append(args, string(*category))
		nextArg++
	}
	if strings.TrimSpace(tag) != "" {
		filters = append(filters, fmt.Sprintf("$%d = any(e.tags)", nextArg))
		args = append(args, tag)
		nextArg++
	}

	args = append(args, limit)
	sql := selectThoughtSQL() + " where " + strings.Join(filters, " and ") + fmt.Sprintf(" order by t.created_at desc limit $%d", nextArg)
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, scanThought)
}

func (r *ThoughtRepository) NextPending(ctx context.Context) (*domain.Thought, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
select id, user_id, created_at, updated_at, audio_s3_key, mime_type,
    duration_ms, size_bytes, transcript, status, attempts, last_error,
    transcribed_at, null::text as category, null::text[] as tags, null::text as title,
    null::text as summary, null::text as model, null::text as prompt_version
from thoughts
where status = 'pending'
order by created_at asc
for update skip locked
limit 1`)
	if err != nil {
		return nil, err
	}
	thought, err := pgx.CollectOneRow(rows, scanThought)
	rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, tx.Commit(ctx)
		}
		return nil, err
	}
	if _, err := tx.Exec(ctx, "update thoughts set status = 'transcribing', updated_at = now() where id = $1", thought.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &thought, nil
}

func (r *ThoughtRepository) NextEnriching(ctx context.Context) (*domain.Thought, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
select id, user_id, created_at, updated_at, audio_s3_key, mime_type,
    duration_ms, size_bytes, transcript, status, attempts, last_error,
    transcribed_at, null::text as category, null::text[] as tags, null::text as title,
    null::text as summary, null::text as model, null::text as prompt_version
from thoughts
where status = 'enriching'
    and transcript is not null
    and not exists (select 1 from thought_enrichments where thought_enrichments.thought_id = thoughts.id)
order by created_at asc
for update skip locked
limit 1`)
	if err != nil {
		return nil, err
	}
	thought, err := pgx.CollectOneRow(rows, scanThought)
	rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, tx.Commit(ctx)
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &thought, nil
}

func (r *ThoughtRepository) MarkTranscribed(ctx context.Context, id uuid.UUID, transcript string) error {
	_, err := r.pool.Exec(ctx, `
update thoughts
set transcript = $1, status = 'enriching', attempts = 0, last_error = null,
    transcribed_at = now(), updated_at = now()
where id = $2`, transcript, id)
	return err
}

func (r *ThoughtRepository) MarkEnriched(ctx context.Context, id uuid.UUID, enrichment domain.ThoughtEnrichment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
insert into thought_enrichments (thought_id, category, tags, title, summary, model, prompt_version)
values ($1, $2, $3, $4, $5, $6, $7)`,
		id, enrichment.Category, enrichment.Tags, enrichment.Title, enrichment.Summary, enrichment.Model, enrichment.PromptVersion,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "update thoughts set status = 'done', updated_at = now() where id = $1", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ThoughtRepository) RecordFailure(ctx context.Context, id uuid.UUID, status domain.Status, attempts int, message string) error {
	_, err := r.pool.Exec(ctx, `
update thoughts
set status = $1, attempts = $2, last_error = $3, last_attempt_at = now(), updated_at = now()
where id = $4`, status, attempts, truncate(message, 1000), id)
	return err
}

func (r *ThoughtRepository) RecordRetry(ctx context.Context, id uuid.UUID, status domain.Status, attempts int, message string) error {
	_, err := r.pool.Exec(ctx, `
update thoughts
set status = $1, attempts = $2, last_error = $3, last_attempt_at = now(), updated_at = now()
where id = $4`, status, attempts, truncate(message, 1000), id)
	return err
}

func (r *ThoughtRepository) RecoverStuckRows(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
update thoughts
set status = case when status = 'transcribing' then 'pending' else status end,
    updated_at = now()
where status in ('transcribing', 'enriching')`)
	return err
}

func selectThoughtSQL() string {
	return `
select t.id, t.user_id, t.created_at, t.updated_at, t.audio_s3_key, t.mime_type,
    t.duration_ms, t.size_bytes, t.transcript, t.status, t.attempts, t.last_error,
    t.transcribed_at, e.category, e.tags, e.title, e.summary, e.model, e.prompt_version
from thoughts t
left join thought_enrichments e on e.thought_id = t.id`
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func scanThought(row pgx.CollectableRow) (domain.Thought, error) {
	var thought domain.Thought
	var category *string
	var tags []string
	var title *string
	var summary *string
	var model *string
	var promptVersion *string
	err := row.Scan(
		&thought.ID,
		&thought.UserID,
		&thought.CreatedAt,
		&thought.UpdatedAt,
		&thought.AudioS3Key,
		&thought.MimeType,
		&thought.DurationMs,
		&thought.SizeBytes,
		&thought.Transcript,
		&thought.Status,
		&thought.Attempts,
		&thought.LastError,
		&thought.TranscribedAt,
		&category,
		&tags,
		&title,
		&summary,
		&model,
		&promptVersion,
	)
	if err != nil {
		return domain.Thought{}, err
	}
	if category != nil && title != nil && summary != nil && model != nil && promptVersion != nil {
		thought.Enrichment = &domain.ThoughtEnrichment{
			Category:      domain.Category(*category),
			Tags:          tags,
			Title:         *title,
			Summary:       *summary,
			Model:         *model,
			PromptVersion: *promptVersion,
		}
	}
	return thought, nil
}
