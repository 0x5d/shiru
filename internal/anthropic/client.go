package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

//go:generate go run go.uber.org/mock/mockgen -destination mock/client.go -package mock . Client

type Client interface {
	GenerateTags(ctx context.Context, surface string) ([]string, error)
	GenerateTopics(ctx context.Context, tags []string, jlptLevel string) ([]string, error)
	RankTags(ctx context.Context, topic string, tags []string) ([]string, error)
	GenerateStory(ctx context.Context, params StoryParams) (*StoryResult, error)
}

type StoryParams struct {
	Topic           string
	Vocab           []string
	JLPTLevel       string
	Tone            string
	TargetWordCount int
}

type StoryResult struct {
	Title string `json:"title"`
	Story string `json:"story"`
}

var _ Client = (*client)(nil)

type messagesAPI interface {
	New(ctx context.Context, params anthropicsdk.MessageNewParams, opts ...option.RequestOption) (*anthropicsdk.Message, error)
}

type client struct {
	messages messagesAPI
	model    anthropicsdk.Model
}

func New(apiKey, model string) *client {
	sdk := anthropicsdk.NewClient(option.WithAPIKey(apiKey))
	return &client{
		messages: &sdk.Messages,
		model:    anthropicsdk.Model(model),
	}
}

func (c *client) GenerateTags(ctx context.Context, surface string) ([]string, error) {
	msg, err := c.messages.New(ctx, anthropicsdk.MessageNewParams{
		Model:     c.model,
		MaxTokens: 256,
		System: []anthropicsdk.TextBlockParam{
			{Text: tagGenerationSystemPrompt},
		},
		Messages: []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(surface)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic for tag generation: %w", err)
	}

	text, err := extractText(msg)
	if err != nil {
		return nil, fmt.Errorf("extracting tag response: %w", err)
	}

	var result tagsResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parsing tag response: %w", err)
	}

	if len(result.Tags) == 0 || len(result.Tags) > 3 {
		return nil, fmt.Errorf("unexpected tag count: %d", len(result.Tags))
	}

	return result.Tags, nil
}

func (c *client) GenerateTopics(ctx context.Context, tags []string, jlptLevel string) ([]string, error) {
	prompt := fmt.Sprintf("User's vocabulary tags: %s\nJLPT level: %s", strings.Join(tags, ", "), jlptLevel)

	msg, err := c.messages.New(ctx, anthropicsdk.MessageNewParams{
		Model:     c.model,
		MaxTokens: 512,
		System: []anthropicsdk.TextBlockParam{
			{Text: topicGenerationSystemPrompt},
		},
		Messages: []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic for topic generation: %w", err)
	}

	text, err := extractText(msg)
	if err != nil {
		return nil, fmt.Errorf("extracting topic response: %w", err)
	}

	var result topicsResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parsing topic response: %w", err)
	}

	if len(result.Topics) != 3 {
		return nil, fmt.Errorf("expected 3 topics, got %d", len(result.Topics))
	}

	return result.Topics, nil
}

func (c *client) RankTags(ctx context.Context, topic string, tags []string) ([]string, error) {
	prompt := fmt.Sprintf("Topic: %s\nAvailable tags: %s", topic, strings.Join(tags, ", "))

	msg, err := c.messages.New(ctx, anthropicsdk.MessageNewParams{
		Model:     c.model,
		MaxTokens: 256,
		System: []anthropicsdk.TextBlockParam{
			{Text: tagRankingSystemPrompt},
		},
		Messages: []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic for tag ranking: %w", err)
	}

	text, err := extractText(msg)
	if err != nil {
		return nil, fmt.Errorf("extracting tag ranking response: %w", err)
	}

	var result topTagsResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parsing tag ranking response: %w", err)
	}

	if len(result.TopTags) == 0 || len(result.TopTags) > 3 {
		return nil, fmt.Errorf("unexpected ranked tag count: %d", len(result.TopTags))
	}

	return result.TopTags, nil
}

func (c *client) GenerateStory(ctx context.Context, params StoryParams) (*StoryResult, error) {
	prompt := fmt.Sprintf(
		"Topic: %s\nTone: %s\nJLPT Level: %s\nTarget word count: %d\nVocabulary to use: %s",
		params.Topic, params.Tone, params.JLPTLevel, params.TargetWordCount,
		strings.Join(params.Vocab, "、"),
	)

	msg, err := c.messages.New(ctx, anthropicsdk.MessageNewParams{
		Model:     c.model,
		MaxTokens: 4096,
		System: []anthropicsdk.TextBlockParam{
			{Text: storyGenerationSystemPrompt},
		},
		Messages: []anthropicsdk.MessageParam{
			anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic for story generation: %w", err)
	}

	text, err := extractText(msg)
	if err != nil {
		return nil, fmt.Errorf("extracting story response: %w", err)
	}

	var result StoryResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parsing story response: %w", err)
	}

	if result.Title == "" || result.Story == "" {
		return nil, fmt.Errorf("story response missing title or story content")
	}

	return &result, nil
}

func extractText(msg *anthropicsdk.Message) (string, error) {
	if len(msg.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic")
	}
	return msg.Content[0].Text, nil
}

type tagsResponse struct {
	Tags []string `json:"tags"`
}

type topicsResponse struct {
	Topics []string `json:"topics"`
}

type topTagsResponse struct {
	TopTags []string `json:"top_tags"`
}
