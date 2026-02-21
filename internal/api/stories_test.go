package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0x5d/shiru/internal/domain"
	domainmock "github.com/0x5d/shiru/internal/domain/mock"
	"github.com/0x5d/shiru/internal/elasticsearch"
	esmock "github.com/0x5d/shiru/internal/elasticsearch/mock"
	"github.com/0x5d/shiru/internal/story"
	storymock "github.com/0x5d/shiru/internal/story/mock"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSearchStories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		url        string
		wantStatus int
		setup      func(*esmock.MockClient)
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "missing query",
			url:        "/api/v1/stories/search",
			wantStatus: http.StatusBadRequest,
			setup:      func(_ *esmock.MockClient) {},
		},
		{
			name:       "successful search",
			url:        "/api/v1/stories/search?q=花&limit=10",
			wantStatus: http.StatusOK,
			setup: func(es *esmock.MockClient) {
				es.EXPECT().
					SearchStories(gomock.Any(), domain.DefaultUserID.String(), "花", 10, 0).
					Return([]elasticsearch.SearchResult{
						{
							StoryID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
							Topic:     "夏祭り",
							Tone:      "funny",
							Content:   "花がきれい",
							JLPTLevel: "N4",
							CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					}, 1, nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp searchStoriesResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, 1, resp.Total)
				require.Len(t, resp.Results, 1)
				assert.Equal(t, "夏祭り", resp.Results[0].Topic)
			},
		},
		{
			name:       "es error",
			url:        "/api/v1/stories/search?q=test",
			wantStatus: http.StatusInternalServerError,
			setup: func(es *esmock.MockClient) {
				es.EXPECT().
					SearchStories(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, 0, fmt.Errorf("es error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			es := esmock.NewMockClient(ctrl)
			tt.setup(es)

			srv := NewServer(logr.Discard(), domainmock.NewMockSettingsRepository(ctrl), domainmock.NewMockVocabRepository(ctrl), nil, nil, nil, nil, es, nil, nil, nil, nil, nil, "")
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

func TestGetStoryTokens(t *testing.T) {
	t.Parallel()

	storyID := uuid.New()
	vocabID := uuid.New()

	tests := []struct {
		name       string
		url        string
		wantStatus int
		setup      func(*storymock.MockRepository, *esmock.MockClient, *domainmock.MockVocabRepository)
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "invalid story ID",
			url:        "/api/v1/stories/bad-id/tokens",
			wantStatus: http.StatusBadRequest,
			setup:      func(_ *storymock.MockRepository, _ *esmock.MockClient, _ *domainmock.MockVocabRepository) {},
		},
		{
			name:       "story not found",
			url:        fmt.Sprintf("/api/v1/stories/%s/tokens", storyID),
			wantStatus: http.StatusNotFound,
			setup: func(sr *storymock.MockRepository, _ *esmock.MockClient, _ *domainmock.MockVocabRepository) {
				sr.EXPECT().Get(gomock.Any(), storyID).Return(nil, story.ErrNotFound)
			},
		},
		{
			name:       "successful tokenization with vocab match",
			url:        fmt.Sprintf("/api/v1/stories/%s/tokens", storyID),
			wantStatus: http.StatusOK,
			setup: func(sr *storymock.MockRepository, es *esmock.MockClient, vr *domainmock.MockVocabRepository) {
				sr.EXPECT().Get(gomock.Any(), storyID).Return(&story.Story{
					ID:      storyID,
					UserID:  domain.DefaultUserID,
					Content: "花がきれい",
				}, nil)

				es.EXPECT().Tokenize(gomock.Any(), "花がきれい").Return([]elasticsearch.Token{
					{Surface: "花", StartOffset: 0, EndOffset: 1, Position: 0},
					{Surface: "が", StartOffset: 1, EndOffset: 2, Position: 1},
					{Surface: "きれい", StartOffset: 2, EndOffset: 5, Position: 2},
				}, nil)

				vr.EXPECT().GetByNormalizedSurfaces(gomock.Any(), domain.DefaultUserID, gomock.Any()).Return([]domain.VocabEntry{
					{ID: vocabID, NormalizedSurface: "花"},
				}, nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp storyTokensResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, storyID, resp.StoryID)
				require.Len(t, resp.Tokens, 3)
				assert.True(t, resp.Tokens[0].IsVocabMatch)
				assert.Equal(t, vocabID, *resp.Tokens[0].VocabEntryID)
				assert.False(t, resp.Tokens[1].IsVocabMatch)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sr := storymock.NewMockRepository(ctrl)
			es := esmock.NewMockClient(ctrl)
			vr := domainmock.NewMockVocabRepository(ctrl)
			tt.setup(sr, es, vr)

			srv := NewServer(logr.Discard(), domainmock.NewMockSettingsRepository(ctrl), vr, sr, nil, nil, nil, es, nil, nil, nil, nil, nil, "")
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
