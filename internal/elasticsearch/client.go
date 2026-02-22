package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
)

const IndexName = "stories_v2"

type StoryDocument struct {
	StoryID   string    `json:"story_id"`
	UserID    string    `json:"user_id"`
	Topic     string    `json:"topic"`
	Tone      string    `json:"tone"`
	Content   string    `json:"content"`
	JLPTLevel string    `json:"jlpt_level"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchResult struct {
	StoryID   uuid.UUID
	Topic     string
	Tone      string
	Content   string
	JLPTLevel string
	CreatedAt time.Time
}

type Token struct {
	Surface       string `json:"token"`
	StartOffset   int    `json:"start_offset"`
	EndOffset     int    `json:"end_offset"`
	Type          string `json:"type"`
	Position      int    `json:"position"`
	Reading       string `json:"reading,omitempty"`
	PartOfSpeech  string `json:"partOfSpeech,omitempty"`
}

//go:generate go run go.uber.org/mock/mockgen -destination mock/client.go -package mock . Client

type Client interface {
	EnsureIndex(ctx context.Context) error
	IndexStory(ctx context.Context, doc StoryDocument) error
	SearchStories(ctx context.Context, userID string, query string, limit, offset int) ([]SearchResult, int, error)
	Tokenize(ctx context.Context, text string) ([]Token, error)
}

var _ Client = (*esClient)(nil)

type esClient struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *esClient {
	return &esClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *esClient) EnsureIndex(ctx context.Context) error {
	mapping := map[string]any{
		"settings": map[string]any{
			"analysis": map[string]any{
				"analyzer": map[string]any{
					"ja_analyzer": map[string]any{
						"type":      "custom",
						"tokenizer": "kuromoji_tokenizer",
						"filter":    []string{"kuromoji_baseform", "kuromoji_part_of_speech", "cjk_width", "lowercase"},
					},
				},
			},
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"story_id":   map[string]any{"type": "keyword"},
				"user_id":    map[string]any{"type": "keyword"},
				"topic":      map[string]any{"type": "text", "analyzer": "ja_analyzer"},
				"tone":       map[string]any{"type": "keyword"},
				"content":    map[string]any{"type": "text", "analyzer": "ja_analyzer"},
				"jlpt_level": map[string]any{"type": "keyword"},
				"created_at": map[string]any{"type": "date"},
			},
		},
	}

	body, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("marshaling index mapping: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/"+IndexName, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating index request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	var esResp struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&esResp)
	if esResp.Error.Type == "resource_already_exists_exception" {
		return nil
	}

	return fmt.Errorf("creating index: status %d, error: %s", resp.StatusCode, esResp.Error.Type)
}

func (c *esClient) IndexStory(ctx context.Context, doc StoryDocument) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling story document: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, IndexName, doc.StoryID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating index request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("indexing story: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("indexing story: unexpected status %d", resp.StatusCode)
	}

	return nil
}

func (c *esClient) SearchStories(ctx context.Context, userID string, query string, limit, offset int) ([]SearchResult, int, error) {
	searchQuery := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"term": map[string]any{
							"user_id": userID,
						},
					},
					map[string]any{
						"multi_match": map[string]any{
							"query":  query,
							"fields": []string{"content", "topic"},
						},
					},
				},
			},
		},
		"from": offset,
		"size": limit,
		"sort": []any{
			map[string]any{"_score": map[string]any{"order": "desc"}},
			map[string]any{"created_at": map[string]any{"order": "desc"}},
		},
	}

	body, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("marshaling search query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", c.baseURL, IndexName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("creating search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("searching stories: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, 0, fmt.Errorf("searching stories: unexpected status %d", resp.StatusCode)
	}

	var esResp struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source StoryDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, 0, fmt.Errorf("decoding search response: %w", err)
	}

	results := make([]SearchResult, len(esResp.Hits.Hits))
	for i, hit := range esResp.Hits.Hits {
		id, _ := uuid.Parse(hit.Source.StoryID)
		results[i] = SearchResult{
			StoryID:   id,
			Topic:     hit.Source.Topic,
			Tone:      hit.Source.Tone,
			Content:   hit.Source.Content,
			JLPTLevel: hit.Source.JLPTLevel,
			CreatedAt: hit.Source.CreatedAt,
		}
	}

	return results, esResp.Hits.Total.Value, nil
}

func (c *esClient) Tokenize(ctx context.Context, text string) ([]Token, error) {
	analyzeReq := map[string]any{
		"analyzer": "ja_analyzer",
		"text":     text,
		"explain":  true,
	}

	body, err := json.Marshal(analyzeReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling analyze request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_analyze", c.baseURL, IndexName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating analyze request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tokenizing text: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("tokenizing text: unexpected status %d", resp.StatusCode)
	}

	var esResp struct {
		Detail struct {
			Tokenizer struct {
				Tokens []Token `json:"tokens"`
			} `json:"tokenizer"`
		} `json:"detail"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("decoding analyze response: %w", err)
	}

	tokens := removeOverlapping(esResp.Detail.Tokenizer.Tokens)
	for i := range tokens {
		tokens[i].Reading = katakanaToHiragana(tokens[i].Reading)
	}

	return tokens, nil
}

// removeOverlapping filters out sub-tokens produced by Kuromoji's search-mode
// compound decomposition. When a compound word like 放課後 is decomposed, the
// tokenizer emits both the original compound and its parts at overlapping
// offsets. We keep only non-overlapping tokens by skipping any token whose
// start offset falls before the end offset of the previously accepted token.
func removeOverlapping(tokens []Token) []Token {
	if len(tokens) == 0 {
		return tokens
	}
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].StartOffset != tokens[j].StartOffset {
			return tokens[i].StartOffset < tokens[j].StartOffset
		}
		return tokens[i].EndOffset > tokens[j].EndOffset
	})
	result := make([]Token, 0, len(tokens))
	end := 0
	for _, t := range tokens {
		if t.StartOffset >= end {
			result = append(result, t)
			end = t.EndOffset
		}
	}
	return result
}

func katakanaToHiragana(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 0x30A1 && r <= 0x30F6 {
			r -= 0x60
		}
		result = append(result, r)
	}
	return string(result)
}
