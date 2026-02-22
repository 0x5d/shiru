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

	"github.com/0x5d/shiru/internal/dictionary"
	dictmock "github.com/0x5d/shiru/internal/dictionary/mock"
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

	srv := NewServer(context.Background(), logr.Discard(), settingsRepo, vocabRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "")
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

	srv := NewServer(context.Background(), logr.Discard(), settingsRepo, vocabRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "")
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

			srv := NewServer(context.Background(), logr.Discard(), settingsRepo, vocabRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "")
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

	srv := NewServer(context.Background(), logr.Discard(), settingsRepo, vocabRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vocab", strings.NewReader(`{"entries":["花","花"]}`))
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp createVocabResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Entries, 1)
	assert.Equal(t, id1, resp.Entries[0].ID)
}

func TestGetVocabDetails(t *testing.T) {
	t.Parallel()

	vocabID := uuid.New()
	meaning := "flower"
	reading := "はな"

	tests := []struct {
		name       string
		url        string
		wantStatus int
		setup      func(*mock.MockVocabRepository, *dictmock.MockClient)
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "invalid vocab ID",
			url:        "/api/v1/vocab/bad-id/details",
			wantStatus: http.StatusBadRequest,
			setup:      func(_ *mock.MockVocabRepository, _ *dictmock.MockClient) {},
		},
		{
			name:       "not found",
			url:        fmt.Sprintf("/api/v1/vocab/%s/details", vocabID),
			wantStatus: http.StatusNotFound,
			setup: func(vr *mock.MockVocabRepository, _ *dictmock.MockClient) {
				vr.EXPECT().GetByID(gomock.Any(), vocabID).Return(nil, domain.ErrVocabNotFound)
			},
		},
		{
			name:       "cached details",
			url:        fmt.Sprintf("/api/v1/vocab/%s/details", vocabID),
			wantStatus: http.StatusOK,
			setup: func(vr *mock.MockVocabRepository, _ *dictmock.MockClient) {
				vr.EXPECT().GetByID(gomock.Any(), vocabID).Return(&domain.VocabEntry{
					ID:      vocabID,
					Surface: "花",
					Meaning: &meaning,
					Reading: &reading,
				}, nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp vocabDetailsResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, "花", resp.Surface)
				assert.Equal(t, "flower", resp.Meaning)
				assert.Equal(t, "はな", resp.Reading)
			},
		},
		{
			name:       "dictionary lookup on miss",
			url:        fmt.Sprintf("/api/v1/vocab/%s/details", vocabID),
			wantStatus: http.StatusOK,
			setup: func(vr *mock.MockVocabRepository, dc *dictmock.MockClient) {
				vr.EXPECT().GetByID(gomock.Any(), vocabID).Return(&domain.VocabEntry{
					ID:      vocabID,
					Surface: "花",
				}, nil)

				dc.EXPECT().Lookup(gomock.Any(), "花").Return(&dictionary.Result{
					Meaning: "flower; blossom",
					Reading: "はな",
				}, nil)

				vr.EXPECT().UpdateDetails(gomock.Any(), vocabID, "flower; blossom", "はな").Return(nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp vocabDetailsResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, "flower; blossom", resp.Meaning)
				assert.Equal(t, "はな", resp.Reading)
			},
		},
		{
			name:       "dictionary error still returns entry",
			url:        fmt.Sprintf("/api/v1/vocab/%s/details", vocabID),
			wantStatus: http.StatusOK,
			setup: func(vr *mock.MockVocabRepository, dc *dictmock.MockClient) {
				vr.EXPECT().GetByID(gomock.Any(), vocabID).Return(&domain.VocabEntry{
					ID:      vocabID,
					Surface: "花",
				}, nil)

				dc.EXPECT().Lookup(gomock.Any(), "花").Return(nil, fmt.Errorf("api error"))
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp vocabDetailsResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, "花", resp.Surface)
				assert.Equal(t, "", resp.Meaning)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			vr := mock.NewMockVocabRepository(ctrl)
			dc := dictmock.NewMockClient(ctrl)
			tt.setup(vr, dc)

			srv := NewServer(context.Background(), logr.Discard(), mock.NewMockSettingsRepository(ctrl), vr, nil, nil, nil, nil, nil, dc, nil, nil, nil, nil, nil, "")
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.check != nil {
				tt.check(t, w)
			}
		})
	}
}
