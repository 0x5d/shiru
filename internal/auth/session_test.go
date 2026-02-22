package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "this-is-a-test-secret-key-that-is-at-least-32-bytes-long"

func newTestSessionManager(t *testing.T, ttl time.Duration) *SessionManager {
	t.Helper()
	sm, err := NewSessionManager(testSecret, ttl)
	require.NoError(t, err)
	return sm
}

func TestSessionManager_RoundTrip(t *testing.T) {
	t.Parallel()

	sm := newTestSessionManager(t, 24*time.Hour)
	userID := uuid.New()

	cookie, err := sm.Encode(userID)
	require.NoError(t, err)
	require.NotEmpty(t, cookie)

	payload, err := sm.Decode(cookie)
	require.NoError(t, err)
	assert.Equal(t, userID, payload.UserID)
	assert.Greater(t, payload.Exp, time.Now().Unix())
}

func TestSessionManager_ExpiredSession(t *testing.T) {
	t.Parallel()

	sm := newTestSessionManager(t, -1*time.Hour)
	userID := uuid.New()

	cookie, err := sm.Encode(userID)
	require.NoError(t, err)

	_, err = sm.Decode(cookie)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session expired")
}

func TestSessionManager_TamperedPayload(t *testing.T) {
	t.Parallel()

	sm := newTestSessionManager(t, 24*time.Hour)
	userID := uuid.New()

	cookie, err := sm.Encode(userID)
	require.NoError(t, err)

	parts := strings.SplitN(cookie, ".", 2)
	require.Len(t, parts, 2)

	// Tamper with the payload by changing the user ID
	tamperedPayload := SessionPayload{
		UserID: uuid.New(),
		Exp:    time.Now().Add(24 * time.Hour).Unix(),
	}
	tamperedBytes, err := json.Marshal(tamperedPayload)
	require.NoError(t, err)
	tamperedB64 := base64.RawURLEncoding.EncodeToString(tamperedBytes)

	tamperedCookie := tamperedB64 + "." + parts[1]
	_, err = sm.Decode(tamperedCookie)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid session signature")
}

func TestSessionManager_TamperedSignature(t *testing.T) {
	t.Parallel()

	sm := newTestSessionManager(t, 24*time.Hour)
	userID := uuid.New()

	cookie, err := sm.Encode(userID)
	require.NoError(t, err)

	parts := strings.SplitN(cookie, ".", 2)
	require.Len(t, parts, 2)

	// Replace signature with garbage
	tamperedCookie := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))
	_, err = sm.Decode(tamperedCookie)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid session signature")
}

func TestSessionManager_WrongSecret(t *testing.T) {
	t.Parallel()

	sm1, err := NewSessionManager("secret-one-that-is-at-least-thirty-two-bytes", 24*time.Hour)
	require.NoError(t, err)
	sm2, err := NewSessionManager("secret-two-that-is-at-least-thirty-two-bytes", 24*time.Hour)
	require.NoError(t, err)

	cookie, err := sm1.Encode(uuid.New())
	require.NoError(t, err)

	_, err = sm2.Decode(cookie)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid session signature")
}

func TestSessionManager_InvalidFormat(t *testing.T) {
	t.Parallel()

	sm := newTestSessionManager(t, 24*time.Hour)

	tests := []struct {
		name   string
		cookie string
	}{
		{"empty string", ""},
		{"no dot separator", "nodot"},
		{"invalid base64 payload", "!!!.valid"},
		{"invalid base64 signature", "dmFsaWQ.!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := sm.Decode(tt.cookie)
			require.Error(t, err)
		})
	}
}

func TestNewSessionManager_ShortSecret(t *testing.T) {
	t.Parallel()

	_, err := NewSessionManager("too-short", 24*time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 bytes")
}

func TestSessionManager_EncodeNilUUID(t *testing.T) {
	t.Parallel()

	sm := newTestSessionManager(t, 24*time.Hour)
	_, err := sm.Encode(uuid.Nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be nil")
}
