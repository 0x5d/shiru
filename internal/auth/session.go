package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SessionPayload struct {
	UserID uuid.UUID `json:"user_id"`
	Exp    int64     `json:"exp"`
}

type SessionManager struct {
	secret []byte
	ttl    time.Duration
}

const minSecretLength = 32

func NewSessionManager(secret string, ttl time.Duration) (*SessionManager, error) {
	if len(secret) < minSecretLength {
		return nil, fmt.Errorf("session secret must be at least %d bytes", minSecretLength)
	}
	return &SessionManager{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

func (m *SessionManager) Encode(userID uuid.UUID) (string, error) {
	if userID == uuid.Nil {
		return "", fmt.Errorf("user ID must not be nil")
	}
	payload := SessionPayload{
		UserID: userID,
		Exp:    time.Now().Add(m.ttl).Unix(),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling session payload: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := m.sign([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return payloadB64 + "." + sigB64, nil
}

func (m *SessionManager) Decode(cookie string) (*SessionPayload, error) {
	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid session format")
	}

	payloadB64 := parts[0]
	sigB64 := parts[1]

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("decoding signature: %w", err)
	}

	expected := m.sign([]byte(payloadB64))
	if !hmac.Equal(sig, expected) {
		return nil, fmt.Errorf("invalid session signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	var payload SessionPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling session payload: %w", err)
	}

	if time.Now().Unix() >= payload.Exp {
		return nil, fmt.Errorf("session expired")
	}

	return &payload, nil
}

func (m *SessionManager) sign(data []byte) []byte {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(data)
	return mac.Sum(nil)
}
