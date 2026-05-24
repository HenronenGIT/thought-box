package repository

import (
	"context"
	"errors"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// UpsertByGoogleSub creates a user when no row matches the given google_sub
// or email, otherwise updates the matching row's email + display name and
// returns it. Email match takes precedence so an existing seeded user is
// adopted on first sign-in (preserving its UUID and any data attached to it).
func (r *UserRepository) UpsertByGoogleSub(ctx context.Context, sub, email, displayName string) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
select id from users
where google_sub = $1
   or (email is not null and lower(email) = lower($2))
limit 1`, sub, email).Scan(&id)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		id = uuid.New()
		if _, err := tx.Exec(ctx, `
insert into users (id, email, google_sub, display_name)
values ($1, $2, $3, $4)`,
			id, email, sub, displayName); err != nil {
			return domain.User{}, err
		}
	case err != nil:
		return domain.User{}, err
	default:
		if _, err := tx.Exec(ctx, `
update users
set email = $1, google_sub = $2, display_name = $3
where id = $4`,
			email, sub, displayName, id); err != nil {
			return domain.User{}, err
		}
	}

	user, err := scanUser(ctx, tx, id)
	if err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// FindByID returns the user row, or (nil, nil) if none exists.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := scanUser(ctx, r.pool, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func scanUser(ctx context.Context, q rowQuerier, id uuid.UUID) (domain.User, error) {
	var user domain.User
	var email, sub, name *string
	err := q.QueryRow(ctx, `
select id, email, google_sub, display_name, created_at
from users where id = $1`, id).Scan(&user.ID, &email, &sub, &name, &user.CreatedAt)
	if err != nil {
		return domain.User{}, err
	}
	if email != nil {
		user.Email = *email
	}
	if sub != nil {
		user.GoogleSub = *sub
	}
	if name != nil {
		user.DisplayName = *name
	}
	return user, nil
}
