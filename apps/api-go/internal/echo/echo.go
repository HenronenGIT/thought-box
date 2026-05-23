package echo

import (
	"context"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
)

type Generator interface {
	Generate(ctx context.Context, transcript string, mode domain.EchoMode) (string, error)
}
