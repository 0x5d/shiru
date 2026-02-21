package dictionary

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Result struct {
	Meaning string
	Reading string
}

//go:generate go run go.uber.org/mock/mockgen -destination mock/client.go -package mock . Client

type Client interface {
	Lookup(ctx context.Context, word string) (*Result, error)
}

var _ Client = (*jishoClient)(nil)

type jishoClient struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *jishoClient {
	if baseURL == "" {
		baseURL = "https://jisho.org/api/v1"
	}
	return &jishoClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *jishoClient) Lookup(ctx context.Context, word string) (*Result, error) {
	reqURL := fmt.Sprintf("%s/search/words?keyword=%s", c.baseURL, url.QueryEscape(word))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating dictionary request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("looking up word %q: %w", word, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dictionary lookup: unexpected status %d", resp.StatusCode)
	}

	var jResp jishoResponse
	if err := json.NewDecoder(resp.Body).Decode(&jResp); err != nil {
		return nil, fmt.Errorf("decoding dictionary response: %w", err)
	}

	if len(jResp.Data) == 0 {
		return nil, fmt.Errorf("no results found for %q", word)
	}

	entry := jResp.Data[0]

	var reading string
	if len(entry.Japanese) > 0 {
		reading = entry.Japanese[0].Reading
	}

	var meanings []string
	for _, sense := range entry.Senses {
		meanings = append(meanings, sense.EnglishDefinitions...)
	}

	return &Result{
		Meaning: strings.Join(meanings, "; "),
		Reading: reading,
	}, nil
}

type jishoResponse struct {
	Data []jishoEntry `json:"data"`
}

type jishoEntry struct {
	Japanese []jishoJapanese `json:"japanese"`
	Senses   []jishoSense    `json:"senses"`
}

type jishoJapanese struct {
	Word    string `json:"word"`
	Reading string `json:"reading"`
}

type jishoSense struct {
	EnglishDefinitions []string `json:"english_definitions"`
}
