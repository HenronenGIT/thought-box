// Package user resolves the caller's user id from request context. The
// session middleware is responsible for putting the id on the context; the
// resolver is just a typed accessor so handlers do not depend on the
// context-key plumbing directly.
package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Resolver interface {
	CurrentUserID(context.Context) (uuid.UUID, error)
}

var ErrNoUser = errors.New("user: no user on context")

type ctxKey int

const userIDKey ctxKey = 1

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func FromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

type ContextResolver struct{}

func (ContextResolver) CurrentUserID(ctx context.Context) (uuid.UUID, error) {
	id, ok := FromContext(ctx)
	if !ok {
		return uuid.Nil, ErrNoUser
	}
	return id, nil
}
