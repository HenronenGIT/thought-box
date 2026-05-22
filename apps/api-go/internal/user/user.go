package user

import (
	"context"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/google/uuid"
)

type Resolver interface {
	CurrentUserID(context.Context) (uuid.UUID, error)
}

type SeededResolver struct{}

func (SeededResolver) CurrentUserID(context.Context) (uuid.UUID, error) {
	return uuid.Parse(domain.SeededUserID)
}
