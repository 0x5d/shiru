package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0x5d/shiru/internal/domain"
	domainmock "github.com/0x5d/shiru/internal/domain/mock"
	"github.com/0x5d/shiru/internal/wanikani"
	wkmock "github.com/0x5d/shiru/internal/wanikani/mock"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestImportWaniKani(t *testing.T) {
	t.Parallel()

	apiKey := "test-api-key"
	lastSynced := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		wantStatus int
		setup      func(*domainmock.MockSettingsRepository, *domainmock.MockVocabRepository, *wkmock.MockClient)
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "successful import with items",
			wantStatus: http.StatusOK,
			setup: func(sr *domainmock.MockSettingsRepository, vr *domainmock.MockVocabRepository, wk *wkmock.MockClient) {
				sr.EXPECT().Get(gomock.Any(), domain.DefaultUserID).Return(&domain.UserSettings{
					WaniKaniAPIKey:       &apiKey,
					WaniKaniLastSyncedAt: &lastSynced,
				}, nil)
				wk.EXPECT().FetchVocabulary(gomock.Any(), apiKey, &lastSynced).Return([]wanikani.VocabItem{
					{SubjectID: 1, Characters: "花"},
					{SubjectID: 2, Characters: "走る"},
				}, nil)
				vr.EXPECT().BatchUpsert(gomock.Any(), domain.DefaultUserID, []string{"花", "走る"}, "wanikani").Return(nil, nil)
				sr.EXPECT().UpdateWaniKaniSyncedAt(gomock.Any(), domain.DefaultUserID, gomock.Any()).Return(nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp importWaniKaniResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, 2, resp.ImportedCount)
			},
		},
		{
			name:       "first sync with no previous timestamp",
			wantStatus: http.StatusOK,
			setup: func(sr *domainmock.MockSettingsRepository, vr *domainmock.MockVocabRepository, wk *wkmock.MockClient) {
				sr.EXPECT().Get(gomock.Any(), domain.DefaultUserID).Return(&domain.UserSettings{
					WaniKaniAPIKey: &apiKey,
				}, nil)
				wk.EXPECT().FetchVocabulary(gomock.Any(), apiKey, (*time.Time)(nil)).Return([]wanikani.VocabItem{
					{SubjectID: 1, Characters: "犬"},
				}, nil)
				vr.EXPECT().BatchUpsert(gomock.Any(), domain.DefaultUserID, []string{"犬"}, "wanikani").Return(nil, nil)
				sr.EXPECT().UpdateWaniKaniSyncedAt(gomock.Any(), domain.DefaultUserID, gomock.Any()).Return(nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp importWaniKaniResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, 1, resp.ImportedCount)
			},
		},
		{
			name:       "no items to import",
			wantStatus: http.StatusOK,
			setup: func(sr *domainmock.MockSettingsRepository, _ *domainmock.MockVocabRepository, wk *wkmock.MockClient) {
				sr.EXPECT().Get(gomock.Any(), domain.DefaultUserID).Return(&domain.UserSettings{
					WaniKaniAPIKey: &apiKey,
				}, nil)
				wk.EXPECT().FetchVocabulary(gomock.Any(), apiKey, (*time.Time)(nil)).Return(nil, nil)
				sr.EXPECT().UpdateWaniKaniSyncedAt(gomock.Any(), domain.DefaultUserID, gomock.Any()).Return(nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp importWaniKaniResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, 0, resp.ImportedCount)
			},
		},
		{
			name:       "no API key configured",
			wantStatus: http.StatusBadRequest,
			setup: func(sr *domainmock.MockSettingsRepository, _ *domainmock.MockVocabRepository, _ *wkmock.MockClient) {
				sr.EXPECT().Get(gomock.Any(), domain.DefaultUserID).Return(&domain.UserSettings{}, nil)
			},
		},
		{
			name:       "empty API key",
			wantStatus: http.StatusBadRequest,
			setup: func(sr *domainmock.MockSettingsRepository, _ *domainmock.MockVocabRepository, _ *wkmock.MockClient) {
				empty := ""
				sr.EXPECT().Get(gomock.Any(), domain.DefaultUserID).Return(&domain.UserSettings{
					WaniKaniAPIKey: &empty,
				}, nil)
			},
		},
		{
			name:       "settings fetch error",
			wantStatus: http.StatusInternalServerError,
			setup: func(sr *domainmock.MockSettingsRepository, _ *domainmock.MockVocabRepository, _ *wkmock.MockClient) {
				sr.EXPECT().Get(gomock.Any(), domain.DefaultUserID).Return(nil, fmt.Errorf("db error"))
			},
		},
		{
			name:       "WaniKani fetch error",
			wantStatus: http.StatusInternalServerError,
			setup: func(sr *domainmock.MockSettingsRepository, _ *domainmock.MockVocabRepository, wk *wkmock.MockClient) {
				sr.EXPECT().Get(gomock.Any(), domain.DefaultUserID).Return(&domain.UserSettings{
					WaniKaniAPIKey: &apiKey,
				}, nil)
				wk.EXPECT().FetchVocabulary(gomock.Any(), apiKey, (*time.Time)(nil)).Return(nil, fmt.Errorf("API error"))
			},
		},
		{
			name:       "batch upsert error",
			wantStatus: http.StatusInternalServerError,
			setup: func(sr *domainmock.MockSettingsRepository, vr *domainmock.MockVocabRepository, wk *wkmock.MockClient) {
				sr.EXPECT().Get(gomock.Any(), domain.DefaultUserID).Return(&domain.UserSettings{
					WaniKaniAPIKey: &apiKey,
				}, nil)
				wk.EXPECT().FetchVocabulary(gomock.Any(), apiKey, (*time.Time)(nil)).Return([]wanikani.VocabItem{
					{SubjectID: 1, Characters: "花"},
				}, nil)
				vr.EXPECT().BatchUpsert(gomock.Any(), domain.DefaultUserID, []string{"花"}, "wanikani").Return(nil, fmt.Errorf("db error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sr := domainmock.NewMockSettingsRepository(ctrl)
			vr := domainmock.NewMockVocabRepository(ctrl)
			wk := wkmock.NewMockClient(ctrl)
			tt.setup(sr, vr, wk)

			srv := NewServer(context.Background(), logr.Discard(), nil, nil, nil, "", 0, false, sr, vr, nil, nil, nil, nil, nil, nil, nil, wk, nil, nil, nil, "")
			req := httptest.NewRequest(http.MethodPost, "/api/v1/vocab/import/wanikani", nil)
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.check != nil {
				tt.check(t, w)
			}
		})
	}
}
