package repository

import (
	"context"
	"errors"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEchoDuplicate  = errors.New("echo mode already exists for thought")
	ErrEchoCapReached = errors.New("echo cap reached for thought")
	ErrThoughtNotReady = errors.New("thought not ready for echoes")
)

type EchoRepository struct {
	pool *pgxpool.Pool
}

func NewEchoRepository(pool *pgxpool.Pool) *EchoRepository {
	return &EchoRepository{pool: pool}
}

func (r *EchoRepository) ListByThought(ctx context.Context, userID uuid.UUID, thoughtID uuid.UUID) ([]domain.Echo, error) {
	rows, err := r.pool.Query(ctx, `
select e.id, e.thought_id, e.mode, e.content, e.status, e.is_default,
       e.attempts, e.last_error, e.model, e.prompt_version, e.created_at, e.updated_at
from echoes e
join thoughts t on t.id = e.thought_id
where t.user_id = $1 and e.thought_id = $2 and e.status <> 'failed'
order by e.is_default desc, e.created_at asc`, userID, thoughtID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, scanEcho)
}

func (r *EchoRepository) RequestEcho(ctx context.Context, userID uuid.UUID, thoughtID uuid.UUID, mode domain.EchoMode) (*domain.Echo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx,
		`select status from thoughts where id = $1 and user_id = $2 for share`,
		thoughtID, userID).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	if status != string(domain.StatusDone) {
		return nil, ErrThoughtNotReady
	}

	var active int
	var sameModeExists bool
	if err := tx.QueryRow(ctx, `
select
  count(*) filter (where status <> 'failed'),
  bool_or(mode = $2 and status <> 'failed')
from echoes
where thought_id = $1`, thoughtID, string(mode)).Scan(&active, &sameModeExists); err != nil {
		return nil, err
	}
	if sameModeExists {
		return nil, ErrEchoDuplicate
	}
	if active >= domain.MaxEchoesPerThought {
		return nil, ErrEchoCapReached
	}

	id := uuid.New()
	rows, err := tx.Query(ctx, `
insert into echoes (id, thought_id, mode, status, is_default)
values ($1, $2, $3, 'pending', false)
returning id, thought_id, mode, content, status, is_default, attempts, last_error, model, prompt_version, created_at, updated_at`,
		id, thoughtID, string(mode))
	if err != nil {
		return nil, err
	}
	echo, err := pgx.CollectOneRow(rows, scanEcho)
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &echo, nil
}

func (r *EchoRepository) InsertDefault(ctx context.Context, id uuid.UUID, thoughtID uuid.UUID, mode domain.EchoMode) error {
	_, err := r.pool.Exec(ctx, `
insert into echoes (id, thought_id, mode, status, is_default)
values ($1, $2, $3, 'pending', true)
on conflict do nothing`, id, thoughtID, string(mode))
	return err
}

type EchoJob struct {
	Echo       domain.Echo
	Transcript string
}

func (r *EchoRepository) NextPending(ctx context.Context) (*EchoJob, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
select e.id, e.thought_id, e.mode, e.content, e.status, e.is_default,
       e.attempts, e.last_error, e.model, e.prompt_version, e.created_at, e.updated_at,
       coalesce(t.transcript, '') as transcript
from echoes e
join thoughts t on t.id = e.thought_id
where e.status = 'pending' and t.transcript is not null
order by e.created_at asc
for update of e skip locked
limit 1`)
	if err != nil {
		return nil, err
	}
	job, err := pgx.CollectOneRow(rows, scanEchoJob)
	rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, tx.Commit(ctx)
		}
		return nil, err
	}
	if _, err := tx.Exec(ctx, "update echoes set status = 'generating', updated_at = now() where id = $1", job.Echo.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *EchoRepository) NextThoughtNeedingDefault(ctx context.Context, allowedCategories []string) (*domain.Thought, error) {
	if len(allowedCategories) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
select t.id, t.user_id, t.created_at, t.updated_at, t.audio_s3_key, t.mime_type,
       t.duration_ms, t.size_bytes, t.transcript, t.status, t.attempts, t.last_error,
       t.transcribed_at, e.category, e.tags, e.title, e.summary, e.model, e.prompt_version
from thoughts t
join thought_enrichments e on e.thought_id = t.id
where t.status = 'done'
  and e.category = any($1)
  and not exists (
    select 1 from echoes ec
    where ec.thought_id = t.id and ec.is_default = true and ec.status <> 'failed'
  )
order by t.updated_at asc
limit 1`, allowedCategories)
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

func (r *EchoRepository) MarkReady(ctx context.Context, id uuid.UUID, content string, model string, promptVersion string) error {
	_, err := r.pool.Exec(ctx, `
update echoes
set status = 'ready', content = $1, model = $2, prompt_version = $3,
    attempts = attempts + 1, last_error = null, last_attempt_at = now(), updated_at = now()
where id = $4`, content, model, promptVersion, id)
	return err
}

func (r *EchoRepository) RecordRetry(ctx context.Context, id uuid.UUID, attempts int, message string) error {
	_, err := r.pool.Exec(ctx, `
update echoes
set status = 'pending', attempts = $1, last_error = $2, last_attempt_at = now(), updated_at = now()
where id = $3`, attempts, truncate(message, 1000), id)
	return err
}

func (r *EchoRepository) RecordFailure(ctx context.Context, id uuid.UUID, attempts int, message string) error {
	_, err := r.pool.Exec(ctx, `
update echoes
set status = 'failed', attempts = $1, last_error = $2, last_attempt_at = now(), updated_at = now()
where id = $3`, attempts, truncate(message, 1000), id)
	return err
}

func (r *EchoRepository) RecoverStuckEchoes(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
update echoes
set status = 'pending', updated_at = now()
where status = 'generating'`)
	return err
}

func scanEcho(row pgx.CollectableRow) (domain.Echo, error) {
	var echo domain.Echo
	var mode, status string
	err := row.Scan(
		&echo.ID,
		&echo.ThoughtID,
		&mode,
		&echo.Content,
		&status,
		&echo.IsDefault,
		&echo.Attempts,
		&echo.LastError,
		&echo.Model,
		&echo.PromptVersion,
		&echo.CreatedAt,
		&echo.UpdatedAt,
	)
	if err != nil {
		return domain.Echo{}, err
	}
	echo.Mode = domain.EchoMode(mode)
	echo.Status = domain.EchoStatus(status)
	return echo, nil
}

func scanEchoJob(row pgx.CollectableRow) (EchoJob, error) {
	var job EchoJob
	var mode, status string
	err := row.Scan(
		&job.Echo.ID,
		&job.Echo.ThoughtID,
		&mode,
		&job.Echo.Content,
		&status,
		&job.Echo.IsDefault,
		&job.Echo.Attempts,
		&job.Echo.LastError,
		&job.Echo.Model,
		&job.Echo.PromptVersion,
		&job.Echo.CreatedAt,
		&job.Echo.UpdatedAt,
		&job.Transcript,
	)
	if err != nil {
		return EchoJob{}, err
	}
	job.Echo.Mode = domain.EchoMode(mode)
	job.Echo.Status = domain.EchoStatus(status)
	return job, nil
}
