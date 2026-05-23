package echo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const Model = openai.ChatModelGPT4oMini

const maxEchoChars = 600

type OpenAIGenerator struct {
	client openai.Client
}

func NewOpenAIGenerator(apiKey string) OpenAIGenerator {
	return OpenAIGenerator{client: openai.NewClient(option.WithAPIKey(apiKey))}
}

func (g OpenAIGenerator) Generate(ctx context.Context, transcript string, mode domain.EchoMode) (string, error) {
	prompt, ok := PromptFor(mode)
	if !ok {
		return "", fmt.Errorf("unsupported echo mode: %s", mode)
	}
	if strings.TrimSpace(transcript) == "" {
		return "", errors.New("empty transcript")
	}

	completion, err := g.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt),
			openai.UserMessage(transcript),
		},
	})
	if err != nil {
		return "", err
	}
	if len(completion.Choices) == 0 {
		return "", errors.New("missing echo choice")
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("empty echo content")
	}
	if len(content) > maxEchoChars {
		content = content[:maxEchoChars]
	}
	return content, nil
}
