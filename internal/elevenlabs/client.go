package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

//go:generate go run go.uber.org/mock/mockgen -destination mock/client.go -package mock . Client

type Client interface {
	GenerateSpeech(ctx context.Context, text string) ([]byte, error)
}

var _ Client = (*elevenlabsClient)(nil)

type elevenlabsClient struct {
	apiKey     string
	voiceID    string
	httpClient *http.Client
}

func New(apiKey, voiceID string) *elevenlabsClient {
	return &elevenlabsClient{
		apiKey:     apiKey,
		voiceID:    voiceID,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type ttsRequest struct {
	Text    string `json:"text"`
	ModelID string `json:"model_id"`
}

func (c *elevenlabsClient) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	body, err := json.Marshal(ttsRequest{
		Text:    text,
		ModelID: "eleven_multilingual_v2",
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling TTS request: %w", err)
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", c.voiceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating TTS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling ElevenLabs TTS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs TTS: status %d: %s", resp.StatusCode, string(respBody))
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading TTS response: %w", err)
	}

	return audio, nil
}
