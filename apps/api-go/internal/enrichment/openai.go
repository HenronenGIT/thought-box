package enrichment

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const (
	PromptVersion = "v1"
	Model         = openai.ChatModelGPT4oMini
)

type OpenAIEnricher struct {
	client openai.Client
}

type structuredEnrichment struct {
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
}

func NewOpenAIEnricher(apiKey string) OpenAIEnricher {
	return OpenAIEnricher{client: openai.NewClient(option.WithAPIKey(apiKey))}
}

func (e OpenAIEnricher) Enrich(ctx context.Context, thoughtID uuid.UUID, transcript string) (domain.ThoughtEnrichment, error) {
	completion, err := e.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("Return concise JSON for a dictated thought. Categories: idea,observation,feeling,learning."),
			openai.UserMessage(transcript),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "thought_enrichment",
					Strict: openai.Bool(true),
					Schema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"category": map[string]any{"type": "string", "enum": []string{"idea", "observation", "feeling", "learning"}},
							"tags":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
							"title":    map[string]any{"type": "string"},
							"summary":  map[string]any{"type": "string"},
						},
						"required":             []string{"category", "tags", "title", "summary"},
						"additionalProperties": false,
					},
				},
			},
		},
	}, option.WithHeader("Idempotency-Key", thoughtID.String()+":"+PromptVersion))
	if err != nil {
		return domain.ThoughtEnrichment{}, err
	}
	if len(completion.Choices) == 0 {
		return domain.ThoughtEnrichment{}, errors.New("missing enrichment choice")
	}
	var parsed structuredEnrichment
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &parsed); err != nil {
		return domain.ThoughtEnrichment{}, err
	}
	category, ok := domain.ParseCategory(parsed.Category)
	if !ok {
		return domain.ThoughtEnrichment{}, errors.New("invalid enrichment category")
	}
	return domain.ThoughtEnrichment{
		Category:      category,
		Tags:          normalizeTags(parsed.Tags),
		Title:         parsed.Title,
		Summary:       parsed.Summary,
		Model:         string(Model),
		PromptVersion: PromptVersion,
	}, nil
}

func normalizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
		if len(out) == 8 {
			break
		}
	}
	return out
}
