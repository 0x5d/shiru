package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0x5d/shiru/internal/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireAuth(t *testing.T) {
	t.Parallel()

	sm := testSessionManager(t)
	srv := newTestServer(sm, nil, nil)

	t.Run("no cookie returns 401", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid cookie returns 401", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
		req.AddCookie(&http.Cookie{Name: "shiru_session", Value: "tampered.value"})
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("expired session returns 401", func(t *testing.T) {
		t.Parallel()
		shortSM, err := auth.NewSessionManager(testSecret, 1*time.Millisecond)
		require.NoError(t, err)

		token, err := shortSM.Encode(uuid.New())
		require.NoError(t, err)

		time.Sleep(5 * time.Millisecond)

		shortSrv := newTestServer(shortSM, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
		req.AddCookie(&http.Cookie{Name: "shiru_session", Value: token})
		w := httptest.NewRecorder()
		shortSrv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("auth endpoints are not protected", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("logout is accessible without auth", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestCORSHeaders(t *testing.T) {
	t.Parallel()

	sm := testSessionManager(t)
	srv := newTestServer(sm, nil, nil)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/settings", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}
