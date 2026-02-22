package anthropic

import (
	"context"
	"fmt"
	"testing"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMessagesAPI struct {
	response *anthropicsdk.Message
	err      error
	captured anthropicsdk.MessageNewParams
}

func (f *fakeMessagesAPI) New(_ context.Context, params anthropicsdk.MessageNewParams, _ ...option.RequestOption) (*anthropicsdk.Message, error) {
	f.captured = params
	return f.response, f.err
}

func newClient(messages messagesAPI, model string) *client {
	return &client{
		messages: messages,
		model:    anthropicsdk.Model(model),
	}
}

func textMessage(text string) *anthropicsdk.Message {
	return &anthropicsdk.Message{
		Content: []anthropicsdk.ContentBlockUnion{
			{Text: text, Type: "text"},
		},
	}
}

func TestGenerateTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		surface  string
		response string
		err      error
		want     []string
		wantErr  string
	}{
		{
			name:     "success with 3 tags",
			surface:  "花",
			response: `{"tags": ["nature", "city", "house"]}`,
			want:     []string{"nature", "city", "house"},
		},
		{
			name:     "success with 1 tag",
			surface:  "猫",
			response: `{"tags": ["animals"]}`,
			want:     []string{"animals"},
		},
		{
			name:     "API error",
			surface:  "花",
			err:      fmt.Errorf("rate limited"),
			wantErr:  "calling Anthropic for tag generation: rate limited",
		},
		{
			name:     "invalid JSON",
			surface:  "花",
			response: `not json`,
			wantErr:  "parsing tag response",
		},
		{
			name:     "too many tags",
			surface:  "花",
			response: `{"tags": ["a", "b", "c", "d"]}`,
			wantErr:  "unexpected tag count: 4",
		},
		{
			name:     "zero tags",
			surface:  "花",
			response: `{"tags": []}`,
			wantErr:  "unexpected tag count: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeMessagesAPI{
				response: textMessage(tt.response),
				err:      tt.err,
			}
			c := newClient(fake, "test-model")

			tags, err := c.GenerateTags(context.Background(), tt.surface)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, tags)
			assert.Contains(t, fake.captured.Messages[0].Content[0].OfText.Text, tt.surface)
		})
	}
}

func TestGenerateTagsBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		surfaces []string
		response string
		err      error
		want     map[string][]string
		wantErr  string
	}{
		{
			name:     "success with multiple words",
			surfaces: []string{"花", "走る"},
			response: `{"results": {"花": ["nature", "city"], "走る": ["exercise", "fitness"]}}`,
			want:     map[string][]string{"花": {"nature", "city"}, "走る": {"exercise", "fitness"}},
		},
		{
			name:     "skips words with invalid tag count",
			surfaces: []string{"花", "走る"},
			response: `{"results": {"花": ["nature"], "走る": ["a", "b", "c", "d"]}}`,
			want:     map[string][]string{"花": {"nature"}},
		},
		{
			name:     "API error",
			surfaces: []string{"花"},
			err:      fmt.Errorf("rate limited"),
			wantErr:  "calling Anthropic for batch tag generation: rate limited",
		},
		{
			name:     "invalid JSON",
			surfaces: []string{"花"},
			response: `not json`,
			wantErr:  "parsing batch tag response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeMessagesAPI{
				response: textMessage(tt.response),
				err:      tt.err,
			}
			c := newClient(fake, "test-model")

			result, err := c.GenerateTagsBatch(context.Background(), tt.surfaces)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGenerateTopics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tags      []string
		jlptLevel string
		response  string
		err       error
		want      []string
		wantErr   string
	}{
		{
			name:      "success",
			tags:      []string{"nature", "food"},
			jlptLevel: "N4",
			response:  `{"topics": ["夏の庭", "料理の冒険", "花見"]}`,
			want:      []string{"夏の庭", "料理の冒険", "花見"},
		},
		{
			name:      "wrong topic count",
			tags:      []string{"nature"},
			jlptLevel: "N5",
			response:  `{"topics": ["one", "two"]}`,
			wantErr:   "expected 3 topics, got 2",
		},
		{
			name:      "API error",
			tags:      []string{"nature"},
			jlptLevel: "N5",
			err:       fmt.Errorf("timeout"),
			wantErr:   "calling Anthropic for topic generation: timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeMessagesAPI{
				response: textMessage(tt.response),
				err:      tt.err,
			}
			c := newClient(fake, "test-model")

			topics, err := c.GenerateTopics(context.Background(), tt.tags, tt.jlptLevel)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, topics)
		})
	}
}

func TestRankTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		topic    string
		tags     []string
		response string
		err      error
		want     []string
		wantErr  string
	}{
		{
			name:     "success",
			topic:    "夏祭り",
			tags:     []string{"nature", "food", "technology", "family"},
			response: `{"top_tags": ["nature", "food", "family"]}`,
			want:     []string{"nature", "food", "family"},
		},
		{
			name:     "fewer than 3 tags",
			topic:    "猫",
			tags:     []string{"animals"},
			response: `{"top_tags": ["animals"]}`,
			want:     []string{"animals"},
		},
		{
			name:     "API error",
			topic:    "猫",
			tags:     []string{"animals"},
			err:      fmt.Errorf("bad request"),
			wantErr:  "calling Anthropic for tag ranking: bad request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeMessagesAPI{
				response: textMessage(tt.response),
				err:      tt.err,
			}
			c := newClient(fake, "test-model")

			ranked, err := c.RankTags(context.Background(), tt.topic, tt.tags)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, ranked)
		})
	}
}

func TestGenerateStory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   StoryParams
		response string
		err      error
		want     *StoryResult
		wantErr  string
	}{
		{
			name: "success",
			params: StoryParams{
				Topic:           "夏祭り",
				Vocab:           []string{"花", "走る", "食べる"},
				JLPTLevel:       "N4",
				Tone:            "funny",
				TargetWordCount: 100,
			},
			response: `{"title": "夏祭りの日", "story": "花がきれいでした。"}`,
			want: &StoryResult{
				Title: "夏祭りの日",
				Story: "花がきれいでした。",
			},
		},
		{
			name: "missing title",
			params: StoryParams{
				Topic:           "猫",
				Vocab:           []string{"猫"},
				JLPTLevel:       "N5",
				Tone:            "shocking",
				TargetWordCount: 50,
			},
			response: `{"title": "", "story": "猫がいます。"}`,
			wantErr:  "story response missing title or story content",
		},
		{
			name: "API error",
			params: StoryParams{
				Topic: "猫",
			},
			err:     fmt.Errorf("server error"),
			wantErr: "calling Anthropic for story generation: server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeMessagesAPI{
				response: textMessage(tt.response),
				err:      tt.err,
			}
			c := newClient(fake, "test-model")

			result, err := c.GenerateStory(context.Background(), tt.params)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestExtractText_EmptyContent(t *testing.T) {
	t.Parallel()

	msg := &anthropicsdk.Message{Content: nil}
	_, err := extractText(msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response from Anthropic")
}
