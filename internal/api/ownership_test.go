package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	audiomock "github.com/0x5d/shiru/internal/audio/mock"
	dictmock "github.com/0x5d/shiru/internal/dictionary/mock"
	"github.com/0x5d/shiru/internal/domain"
	domainmock "github.com/0x5d/shiru/internal/domain/mock"
	esmock "github.com/0x5d/shiru/internal/elasticsearch/mock"
	"github.com/0x5d/shiru/internal/story"
	storymock "github.com/0x5d/shiru/internal/story/mock"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCrossUserStoryAccessDenied(t *testing.T) {
	t.Parallel()

	otherUserStoryID := uuid.New()

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "GET story",
			url:  fmt.Sprintf("/api/v1/stories/%s", otherUserStoryID),
		},
		{
			name: "GET story tokens",
			url:  fmt.Sprintf("/api/v1/stories/%s/tokens", otherUserStoryID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sr := storymock.NewMockRepository(ctrl)
			es := esmock.NewMockClient(ctrl)

			sr.EXPECT().Get(gomock.Any(), testUserID, otherUserStoryID).Return(nil, story.ErrNotFound)

			sm := testSessionManager(t)
			srv := NewServer(context.Background(), logr.Discard(), sm, nil, nil, "shiru_session", 72*time.Hour, false, "http://localhost:5173", domainmock.NewMockSettingsRepository(ctrl), domainmock.NewMockVocabRepository(ctrl), sr, nil, nil, nil, es, nil, nil, nil, nil, nil, nil, "")
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			addAuthCookie(t, sm, req)
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestCrossUserAudioAccessDenied(t *testing.T) {
	t.Parallel()

	otherUserStoryID := uuid.New()
	ctrl := gomock.NewController(t)
	sr := storymock.NewMockRepository(ctrl)

	sr.EXPECT().Get(gomock.Any(), testUserID, otherUserStoryID).Return(nil, story.ErrNotFound)

	sm := testSessionManager(t)
	srv := NewServer(
		context.Background(),
		logr.Discard(),
		sm, nil, nil, "shiru_session", 72*time.Hour, false, "http://localhost:5173",
		domainmock.NewMockSettingsRepository(ctrl),
		domainmock.NewMockVocabRepository(ctrl),
		sr, nil, nil, nil, nil, nil, nil, nil,
		audiomock.NewMockRepository(ctrl),
		audiomock.NewMockFileStore(ctrl),
		nil, "",
	)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/stories/%s/audio", otherUserStoryID), nil)
	addAuthCookie(t, sm, req)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCrossUserVocabAccessDenied(t *testing.T) {
	t.Parallel()

	otherUserVocabID := uuid.New()
	ctrl := gomock.NewController(t)
	vr := domainmock.NewMockVocabRepository(ctrl)
	dc := dictmock.NewMockClient(ctrl)

	vr.EXPECT().GetByID(gomock.Any(), testUserID, otherUserVocabID).Return(nil, domain.ErrVocabNotFound)

	sm := testSessionManager(t)
	srv := NewServer(context.Background(), logr.Discard(), sm, nil, nil, "shiru_session", 72*time.Hour, false, "http://localhost:5173", domainmock.NewMockSettingsRepository(ctrl), vr, nil, nil, nil, nil, nil, dc, nil, nil, nil, nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vocab/%s/details", otherUserVocabID), nil)
	addAuthCookie(t, sm, req)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
