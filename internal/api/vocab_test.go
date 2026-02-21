package api

import (
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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListVocab(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settingsRepo := mock.NewMockSettingsRepository(ctrl)
	vocabRepo := mock.NewMockVocabRepository(ctrl)

	now := time.Now().Truncate(time.Second)
	id := uuid.New()
	vocabRepo.EXPECT().List(gomock.Any(), domain.DefaultUserID, "", 20, 0).Return([]domain.VocabEntry{
		{
			ID:                id,
			UserID:            domain.DefaultUserID,
			Surface:           "花",
			NormalizedSurface: "花",
			Source:            "manual",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}, 1, nil)

	srv := NewServer(logr.Discard(), settingsRepo, vocabRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vocab", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp listVocabResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Entries, 1)
	assert.Equal(t, "花", resp.Entries[0].Surface)
}

func TestListVocabWithQuery(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settingsRepo := mock.NewMockSettingsRepository(ctrl)
	vocabRepo := mock.NewMockVocabRepository(ctrl)

	vocabRepo.EXPECT().List(gomock.Any(), domain.DefaultUserID, "花", 10, 5).Return(nil, 0, nil)

	srv := NewServer(logr.Discard(), settingsRepo, vocabRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vocab?query=花&limit=10&offset=5", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreateVocab(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
		setupMock  func(*mock.MockVocabRepository)
	}{
		{
			name:       "creates entries",
			body:       `{"entries":["花","走る"]}`,
			wantStatus: http.StatusCreated,
			setupMock: func(m *mock.MockVocabRepository) {
				m.EXPECT().BatchUpsert(gomock.Any(), domain.DefaultUserID, []string{"花", "走る"}, "manual").Return([]domain.VocabEntry{
					{ID: uuid.New(), Surface: "花", NormalizedSurface: "花", Source: "manual"},
					{ID: uuid.New(), Surface: "走る", NormalizedSurface: "走る", Source: "manual"},
				}, nil)
			},
		},
		{
			name:       "empty entries",
			body:       `{"entries":[]}`,
			wantStatus: http.StatusBadRequest,
			setupMock:  func(_ *mock.MockVocabRepository) {},
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
			setupMock:  func(_ *mock.MockVocabRepository) {},
		},
		{
			name:       "repo error",
			body:       `{"entries":["花"]}`,
			wantStatus: http.StatusInternalServerError,
			setupMock: func(m *mock.MockVocabRepository) {
				m.EXPECT().BatchUpsert(gomock.Any(), domain.DefaultUserID, []string{"花"}, "manual").Return(nil, fmt.Errorf("db error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			settingsRepo := mock.NewMockSettingsRepository(ctrl)
			vocabRepo := mock.NewMockVocabRepository(ctrl)
			tt.setupMock(vocabRepo)

			srv := NewServer(logr.Discard(), settingsRepo, vocabRepo)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/vocab", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestCreateVocabResponseShape(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	settingsRepo := mock.NewMockSettingsRepository(ctrl)
	vocabRepo := mock.NewMockVocabRepository(ctrl)

	now := time.Now().Truncate(time.Second)
	id1 := uuid.New()
	vocabRepo.EXPECT().BatchUpsert(gomock.Any(), domain.DefaultUserID, []string{"花", "花"}, "manual").Return([]domain.VocabEntry{
		{ID: id1, UserID: domain.DefaultUserID, Surface: "花", NormalizedSurface: "花", Source: "manual", CreatedAt: now, UpdatedAt: now},
	}, nil)

	srv := NewServer(logr.Discard(), settingsRepo, vocabRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vocab", strings.NewReader(`{"entries":["花","花"]}`))
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp createVocabResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Entries, 1)
	assert.Equal(t, id1, resp.Entries[0].ID)
}
