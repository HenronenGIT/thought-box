package enrichment

import (
	"context"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/google/uuid"
)

type Enricher interface {
	Enrich(ctx context.Context, thoughtID uuid.UUID, transcript string) (domain.ThoughtEnrichment, error)
}
