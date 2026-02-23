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

	"github.com/0x5d/shiru/internal/domain"
	"github.com/0x5d/shiru/internal/domain/mock"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetSettings(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settingsRepo := mock.NewMockSettingsRepository(ctrl)
	vocabRepo := mock.NewMockVocabRepository(ctrl)
	sm := testSessionManager(t)

	now := time.Now().Truncate(time.Second)
	settingsRepo.EXPECT().Get(gomock.Any(), testUserID).Return(&domain.UserSettings{
		UserID:          testUserID,
		JLPTLevel:       "N3",
		StoryWordTarget: 150,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil)

	srv := NewServer(context.Background(), logr.Discard(), sm, nil, nil, "shiru_session", 72*time.Hour, false, "http://localhost:5173", settingsRepo, vocabRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	addAuthCookie(t, sm, req)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp settingsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "N3", resp.JLPTLevel)
	assert.Equal(t, 150, resp.StoryWordTarget)
	assert.Nil(t, resp.WaniKaniAPIKey)
}

func TestUpdateSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
		setupMock  func(*mock.MockSettingsRepository)
	}{
		{
			name:       "valid update",
			body:       `{"jlpt_level":"N2","story_word_target":200}`,
			wantStatus: http.StatusOK,
			setupMock: func(m *mock.MockSettingsRepository) {
				m.EXPECT().Update(gomock.Any(), testUserID, "N2", 200, (*string)(nil)).Return(&domain.UserSettings{
					UserID:          testUserID,
					JLPTLevel:       "N2",
					StoryWordTarget: 200,
				}, nil)
			},
		},
		{
			name:       "invalid jlpt level",
			body:       `{"jlpt_level":"N6","story_word_target":100}`,
			wantStatus: http.StatusBadRequest,
			setupMock:  func(_ *mock.MockSettingsRepository) {},
		},
		{
			name:       "story word target too low",
			body:       `{"jlpt_level":"N5","story_word_target":10}`,
			wantStatus: http.StatusBadRequest,
			setupMock:  func(_ *mock.MockSettingsRepository) {},
		},
		{
			name:       "story word target too high",
			body:       `{"jlpt_level":"N5","story_word_target":999}`,
			wantStatus: http.StatusBadRequest,
			setupMock:  func(_ *mock.MockSettingsRepository) {},
		},
		{
			name:       "invalid json",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
			setupMock:  func(_ *mock.MockSettingsRepository) {},
		},
		{
			name:       "repo error",
			body:       `{"jlpt_level":"N1","story_word_target":100}`,
			wantStatus: http.StatusInternalServerError,
			setupMock: func(m *mock.MockSettingsRepository) {
				m.EXPECT().Update(gomock.Any(), testUserID, "N1", 100, (*string)(nil)).Return(nil, fmt.Errorf("db error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			settingsRepo := mock.NewMockSettingsRepository(ctrl)
			vocabRepo := mock.NewMockVocabRepository(ctrl)
			sm := testSessionManager(t)
			tt.setupMock(settingsRepo)

			srv := NewServer(context.Background(), logr.Discard(), sm, nil, nil, "shiru_session", 72*time.Hour, false, "http://localhost:5173", settingsRepo, vocabRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader(tt.body))
			addAuthCookie(t, sm, req)
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
