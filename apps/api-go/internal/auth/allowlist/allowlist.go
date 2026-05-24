// Package allowlist gates which Google-authenticated emails may complete
// sign-in. The list lives in the allowed_emails table so it can be edited
// without redeploying.
package allowlist

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Gate struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Gate {
	return &Gate{pool: pool}
}

// IsAllowed reports whether email appears in allowed_emails. Matching is
// case-insensitive: 'Foo@Bar.com' and 'foo@bar.com' are the same address.
func (g *Gate) IsAllowed(ctx context.Context, email string) (bool, error) {
	var found bool
	err := g.pool.QueryRow(ctx, `
select exists (
  select 1 from allowed_emails where lower(email) = lower($1)
)`, email).Scan(&found)
	return found, err
}
