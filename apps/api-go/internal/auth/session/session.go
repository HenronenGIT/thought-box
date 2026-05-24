// Package session issues, looks up, and revokes opaque server-side sessions.
//
// Tokens are 32 random bytes encoded base64-url for transport (in a cookie).
// Only SHA-256 hashes are stored, so a database leak does not expose live
// sessions. Sliding expiry is implemented by bumping last_seen_at and
// expires_at on each successful Lookup.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const tokenBytes = 32

var (
	ErrNotFound = errors.New("session: not found")
	ErrExpired  = errors.New("session: expired")
)

type Store struct {
	pool *pgxpool.Pool
	ttl  time.Duration
	now  func() time.Time
}

// NewStore wires the store to a pool with default sliding expiry of 30 days
// and the real clock.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, ttl: 30 * 24 * time.Hour, now: time.Now}
}

// NewStoreForTest exposes the sliding-TTL and clock seams so tests can
// exercise expiry and sliding without sleeping for days.
func NewStoreForTest(pool *pgxpool.Pool, ttl time.Duration, now func() time.Time) *Store {
	return &Store{pool: pool, ttl: ttl, now: now}
}

// Issue creates a new session for userID and returns the raw token to set in
// a cookie. ttl bounds the initial expiry; activity will slide it forward.
func (s *Store) Issue(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	raw, err := randomToken()
	if err != nil {
		return "", err
	}
	hash := hashToken(raw)
	now := s.now()
	_, err = s.pool.Exec(ctx, `
insert into sessions (token_hash, user_id, created_at, expires_at, last_seen_at)
values ($1, $2, $3, $4, $3)`,
		hash, userID, now, now.Add(ttl))
	if err != nil {
		return "", err
	}
	return raw, nil
}

// Lookup returns the user id bound to rawToken and bumps last_seen_at +
// expires_at to slide the expiry forward. Returns ErrNotFound for unknown
// tokens, ErrExpired for tokens past their expiry.
func (s *Store) Lookup(ctx context.Context, rawToken string) (uuid.UUID, error) {
	hash := hashToken(rawToken)
	row := s.pool.QueryRow(ctx, `
select user_id, expires_at from sessions where token_hash = $1`, hash)
	var userID uuid.UUID
	var expiresAt time.Time
	if err := row.Scan(&userID, &expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	now := s.now()
	if !now.Before(expiresAt) {
		return uuid.Nil, ErrExpired
	}
	if _, err := s.pool.Exec(ctx, `
update sessions set last_seen_at = $1, expires_at = $2 where token_hash = $3`,
		now, now.Add(s.ttl), hash); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

// Revoke deletes a single session by its raw token. Idempotent: revoking an
// already-deleted or unknown token returns no error.
func (s *Store) Revoke(ctx context.Context, rawToken string) error {
	hash := hashToken(rawToken)
	_, err := s.pool.Exec(ctx, "delete from sessions where token_hash = $1", hash)
	return err
}

// RevokeAll deletes every session for the given user. Used for "log out
// everywhere" and on account-level security events.
func (s *Store) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "delete from sessions where user_id = $1", userID)
	return err
}

func randomToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
