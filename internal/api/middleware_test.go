package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

	t.Run("logout is accessible without auth", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("google login is accessible without auth", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		// Should get 400 (bad request) not 401, proving the endpoint is not behind auth middleware
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAllProtectedEndpointsRequireAuth(t *testing.T) {
	t.Parallel()

	sm := testSessionManager(t)
	srv := newTestServer(sm, nil, nil)

	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/settings"},
		{http.MethodPut, "/api/v1/settings"},
		{http.MethodGet, "/api/v1/vocab"},
		{http.MethodPost, "/api/v1/vocab"},
		{http.MethodGet, "/api/v1/vocab/00000000-0000-0000-0000-000000000001/details"},
		{http.MethodGet, "/api/v1/dictionary/lookup?word=花"},
		{http.MethodGet, "/api/v1/topics"},
		{http.MethodPost, "/api/v1/stories"},
		{http.MethodGet, "/api/v1/stories"},
		{http.MethodGet, "/api/v1/stories/search?q=test"},
		{http.MethodGet, "/api/v1/stories/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/stories/00000000-0000-0000-0000-000000000001/tokens"},
		{http.MethodPost, "/api/v1/stories/00000000-0000-0000-0000-000000000001/audio"},
		{http.MethodPost, "/api/v1/vocab/import/wanikani"},
	}

	for _, ep := range protectedEndpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(ep.method, ep.path, nil)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s %s should return 401 without auth cookie", ep.method, ep.path)
		})
	}
}

func TestCookieFromDifferentSecretRejected(t *testing.T) {
	t.Parallel()

	otherSM, err := auth.NewSessionManager("a-completely-different-secret-key-for-testing!!", 72*time.Hour)
	require.NoError(t, err)

	token, err := otherSM.Encode(uuid.New())
	require.NoError(t, err)

	sm := testSessionManager(t)
	srv := newTestServer(sm, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	req.AddCookie(&http.Cookie{Name: "shiru_session", Value: token})
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCORSHeaders(t *testing.T) {
	t.Parallel()

	sm := testSessionManager(t)
	srv := newTestServer(sm, nil, nil)

	t.Run("matching origin gets CORS headers", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/settings", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Origin", w.Header().Get("Vary"))
	})

	t.Run("non-matching origin does not get CORS allow headers", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/settings", nil)
		req.Header.Set("Origin", "https://evil.com")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Origin", w.Header().Get("Vary"))
	})

	t.Run("no origin header does not get CORS allow headers", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Origin", w.Header().Get("Vary"))
	})
}
