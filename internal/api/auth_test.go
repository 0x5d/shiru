package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0x5d/shiru/internal/auth"
	"github.com/0x5d/shiru/internal/domain"
	"github.com/0x5d/shiru/internal/domain/mock"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testSecret = "test-secret-that-is-at-least-32-bytes-long!!"

func testSessionManager(t *testing.T) *auth.SessionManager {
	t.Helper()
	sm, err := auth.NewSessionManager(testSecret, 72*time.Hour)
	require.NoError(t, err)
	return sm
}

type stubGoogleVerifier struct {
	claims *auth.GoogleClaims
	err    error
}

func (s *stubGoogleVerifier) Verify(_ context.Context, _ string) (*auth.GoogleClaims, error) {
	return s.claims, s.err
}

func newTestServer(sm *auth.SessionManager, gv GoogleTokenVerifier, ur domain.UserRepository) *Server {
	return NewServer(context.Background(), logr.Discard(), sm, gv, ur, "shiru_session", 72*time.Hour, false,
		"http://localhost:5173", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "")
}

func TestGoogleLogin(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	email := "user@example.com"
	name := "Test User"
	avatar := "https://example.com/avatar.png"

	t.Run("missing body", func(t *testing.T) {
		t.Parallel()
		sm := testSessionManager(t)
		srv := newTestServer(sm, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(""))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty credential", func(t *testing.T) {
		t.Parallel()
		sm := testSessionManager(t)
		srv := newTestServer(sm, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(`{"credential":""}`))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid token returns 401 with no session cookie", func(t *testing.T) {
		t.Parallel()
		sm := testSessionManager(t)
		gv := &stubGoogleVerifier{err: assert.AnError}
		srv := newTestServer(sm, gv, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(`{"credential":"bad-token"}`))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assertNoSessionCookie(t, w)
	})

	t.Run("successful login sets cookie and returns user", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		ur := mock.NewMockUserRepository(ctrl)

		ur.EXPECT().UpsertGoogleUser(gomock.Any(), "google-sub-123", email, name, avatar).Return(&domain.User{
			ID:        userID,
			Email:     &email,
			Name:      &name,
			AvatarURL: &avatar,
		}, nil)
		ur.EXPECT().EnsureUserSettings(gomock.Any(), userID).Return(nil)

		sm := testSessionManager(t)
		gv := &stubGoogleVerifier{claims: &auth.GoogleClaims{
			Sub:       "google-sub-123",
			Email:     email,
			Name:      name,
			AvatarURL: avatar,
		}}
		srv := newTestServer(sm, gv, ur)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(`{"credential":"valid-token"}`))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "shiru_session" {
				sessionCookie = c
				break
			}
		}
		require.NotNil(t, sessionCookie)
		assert.True(t, sessionCookie.HttpOnly)
		assert.Equal(t, "/", sessionCookie.Path)
		assert.Equal(t, http.SameSiteLaxMode, sessionCookie.SameSite)
		assert.Equal(t, int(72*time.Hour/time.Second), sessionCookie.MaxAge)

		var resp userResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), resp.ID)
		assert.Equal(t, &email, resp.Email)
		assert.Equal(t, &name, resp.Name)
	})

	t.Run("upsert user failure returns 500", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		ur := mock.NewMockUserRepository(ctrl)

		ur.EXPECT().UpsertGoogleUser(gomock.Any(), "google-sub-123", email, name, avatar).
			Return(nil, fmt.Errorf("db error"))

		sm := testSessionManager(t)
		gv := &stubGoogleVerifier{claims: &auth.GoogleClaims{
			Sub: "google-sub-123", Email: email, Name: name, AvatarURL: avatar,
		}}
		srv := newTestServer(sm, gv, ur)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(`{"credential":"valid-token"}`))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assertNoSessionCookie(t, w)
	})

	t.Run("ensure settings failure returns 500", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		ur := mock.NewMockUserRepository(ctrl)

		ur.EXPECT().UpsertGoogleUser(gomock.Any(), "google-sub-123", email, name, avatar).Return(&domain.User{
			ID: userID, Email: &email, Name: &name, AvatarURL: &avatar,
		}, nil)
		ur.EXPECT().EnsureUserSettings(gomock.Any(), userID).Return(fmt.Errorf("db error"))

		sm := testSessionManager(t)
		gv := &stubGoogleVerifier{claims: &auth.GoogleClaims{
			Sub: "google-sub-123", Email: email, Name: name, AvatarURL: avatar,
		}}
		srv := newTestServer(sm, gv, ur)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google", strings.NewReader(`{"credential":"valid-token"}`))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assertNoSessionCookie(t, w)
	})
}

func TestMe(t *testing.T) {
	t.Parallel()

	t.Run("no cookie returns 401", func(t *testing.T) {
		t.Parallel()
		sm := testSessionManager(t)
		srv := newTestServer(sm, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid cookie returns 401", func(t *testing.T) {
		t.Parallel()
		sm := testSessionManager(t)
		srv := newTestServer(sm, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "shiru_session", Value: "tampered-value"})
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("expired session returns 401", func(t *testing.T) {
		t.Parallel()
		expiredSM, err := auth.NewSessionManager(testSecret, -1*time.Hour)
		require.NoError(t, err)

		userID := uuid.New()
		token, err := expiredSM.Encode(userID)
		require.NoError(t, err)

		srv := newTestServer(expiredSM, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "shiru_session", Value: token})
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid cookie returns user", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		ur := mock.NewMockUserRepository(ctrl)

		userID := uuid.New()
		email := "user@example.com"
		name := "Test User"

		ur.EXPECT().GetByID(gomock.Any(), userID).Return(&domain.User{
			ID:    userID,
			Email: &email,
			Name:  &name,
		}, nil)

		sm := testSessionManager(t)
		srv := newTestServer(sm, nil, ur)

		token, err := sm.Encode(userID)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "shiru_session", Value: token})
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp userResponse
		err = json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), resp.ID)
		assert.Equal(t, &email, resp.Email)
	})

	t.Run("user not found returns 401", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		ur := mock.NewMockUserRepository(ctrl)

		userID := uuid.New()
		ur.EXPECT().GetByID(gomock.Any(), userID).Return(nil, domain.ErrUserNotFound)

		sm := testSessionManager(t)
		srv := newTestServer(sm, nil, ur)

		token, err := sm.Encode(userID)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "shiru_session", Value: token})
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("db error returns 500", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		ur := mock.NewMockUserRepository(ctrl)

		userID := uuid.New()
		ur.EXPECT().GetByID(gomock.Any(), userID).Return(nil, fmt.Errorf("connection refused"))

		sm := testSessionManager(t)
		srv := newTestServer(sm, nil, ur)

		token, err := sm.Encode(userID)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "shiru_session", Value: token})
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()

	sm := testSessionManager(t)
	srv := newTestServer(sm, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "shiru_session" {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, "", sessionCookie.Value)
	assert.True(t, sessionCookie.MaxAge < 0)
	assert.Equal(t, http.SameSiteLaxMode, sessionCookie.SameSite)
}
