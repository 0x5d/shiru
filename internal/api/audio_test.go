package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0x5d/shiru/internal/audio"
	audiomock "github.com/0x5d/shiru/internal/audio/mock"
	"github.com/0x5d/shiru/internal/domain"
	domainmock "github.com/0x5d/shiru/internal/domain/mock"
	"github.com/0x5d/shiru/internal/elevenlabs"
	elevenlabsmock "github.com/0x5d/shiru/internal/elevenlabs/mock"
	"github.com/0x5d/shiru/internal/story"
	storymock "github.com/0x5d/shiru/internal/story/mock"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateStoryAudio(t *testing.T) {
	t.Parallel()

	storyID := uuid.New()

	testVoiceSelector := elevenlabs.NewVoiceSelector("test-voice", nil)

	tests := []struct {
		name       string
		url        string
		wantStatus int
		setup      func(*audiomock.MockRepository, *audiomock.MockFileStore, *storymock.MockRepository, *elevenlabsmock.MockClient, *domainmock.MockSettingsRepository)
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "invalid story ID",
			url:        "/api/v1/stories/bad-id/audio",
			wantStatus: http.StatusBadRequest,
			setup:      func(_ *audiomock.MockRepository, _ *audiomock.MockFileStore, _ *storymock.MockRepository, _ *elevenlabsmock.MockClient, _ *domainmock.MockSettingsRepository) {},
		},
		{
			name:       "cached audio returned",
			url:        fmt.Sprintf("/api/v1/stories/%s/audio", storyID),
			wantStatus: http.StatusOK,
			setup: func(ar *audiomock.MockRepository, fs *audiomock.MockFileStore, sr *storymock.MockRepository, _ *elevenlabsmock.MockClient, _ *domainmock.MockSettingsRepository) {
				sr.EXPECT().Get(gomock.Any(), testUserID, storyID).Return(&story.Story{
					ID:      storyID,
					Content: "花がきれいです。",
					Tone:    "funny",
				}, nil)
				ar.EXPECT().GetByStoryID(gomock.Any(), storyID).Return(&audio.StoryAudio{
					StoryID:     storyID,
					StoragePath: storyID.String() + ".mp3",
				}, nil)
				fs.EXPECT().Read(storyID.String() + ".mp3").Return([]byte("cached-audio-data"), nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "audio/mpeg", w.Header().Get("Content-Type"))
				assert.Equal(t, "cached-audio-data", w.Body.String())
			},
		},
		{
			name:       "generates and caches new audio",
			url:        fmt.Sprintf("/api/v1/stories/%s/audio", storyID),
			wantStatus: http.StatusCreated,
			setup: func(ar *audiomock.MockRepository, fs *audiomock.MockFileStore, sr *storymock.MockRepository, el *elevenlabsmock.MockClient, stg *domainmock.MockSettingsRepository) {
				sr.EXPECT().Get(gomock.Any(), testUserID, storyID).Return(&story.Story{
					ID:      storyID,
					Content: "花がきれいです。",
					Tone:    "funny",
				}, nil)
				stg.EXPECT().Get(gomock.Any(), testUserID).Return(&domain.UserSettings{JLPTLevel: "N5"}, nil)
				ar.EXPECT().GetByStoryID(gomock.Any(), storyID).Return(nil, audio.ErrNotFound)
				el.EXPECT().GenerateSpeech(gomock.Any(), "test-voice", "花がきれいです。").Return([]byte("new-audio-data"), nil)
				fs.EXPECT().Write(storyID.String()+".mp3", []byte("new-audio-data")).Return(nil)
				ar.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "audio/mpeg", w.Header().Get("Content-Type"))
				assert.Equal(t, "new-audio-data", w.Body.String())
			},
		},
		{
			name:       "story not found",
			url:        fmt.Sprintf("/api/v1/stories/%s/audio", storyID),
			wantStatus: http.StatusNotFound,
			setup: func(_ *audiomock.MockRepository, _ *audiomock.MockFileStore, sr *storymock.MockRepository, _ *elevenlabsmock.MockClient, _ *domainmock.MockSettingsRepository) {
				sr.EXPECT().Get(gomock.Any(), testUserID, storyID).Return(nil, story.ErrNotFound)
			},
		},
		{
			name:       "elevenlabs error",
			url:        fmt.Sprintf("/api/v1/stories/%s/audio", storyID),
			wantStatus: http.StatusInternalServerError,
			setup: func(ar *audiomock.MockRepository, _ *audiomock.MockFileStore, sr *storymock.MockRepository, el *elevenlabsmock.MockClient, stg *domainmock.MockSettingsRepository) {
				sr.EXPECT().Get(gomock.Any(), testUserID, storyID).Return(&story.Story{
					ID:      storyID,
					Content: "テスト",
					Tone:    "shocking",
				}, nil)
				stg.EXPECT().Get(gomock.Any(), testUserID).Return(&domain.UserSettings{JLPTLevel: "N4"}, nil)
				ar.EXPECT().GetByStoryID(gomock.Any(), storyID).Return(nil, audio.ErrNotFound)
				el.EXPECT().GenerateSpeech(gomock.Any(), "test-voice", "テスト").Return(nil, fmt.Errorf("TTS error"))
			},
		},
		{
			name:       "audio repo check error",
			url:        fmt.Sprintf("/api/v1/stories/%s/audio", storyID),
			wantStatus: http.StatusInternalServerError,
			setup: func(ar *audiomock.MockRepository, _ *audiomock.MockFileStore, sr *storymock.MockRepository, _ *elevenlabsmock.MockClient, _ *domainmock.MockSettingsRepository) {
				sr.EXPECT().Get(gomock.Any(), testUserID, storyID).Return(&story.Story{
					ID:      storyID,
					Content: "テスト",
					Tone:    "funny",
				}, nil)
				ar.EXPECT().GetByStoryID(gomock.Any(), storyID).Return(nil, fmt.Errorf("db error"))
			},
		},
		{
			name:       "settings error falls back to default voice",
			url:        fmt.Sprintf("/api/v1/stories/%s/audio", storyID),
			wantStatus: http.StatusCreated,
			setup: func(ar *audiomock.MockRepository, fs *audiomock.MockFileStore, sr *storymock.MockRepository, el *elevenlabsmock.MockClient, stg *domainmock.MockSettingsRepository) {
				sr.EXPECT().Get(gomock.Any(), testUserID, storyID).Return(&story.Story{
					ID:      storyID,
					Content: "テスト",
					Tone:    "funny",
				}, nil)
				ar.EXPECT().GetByStoryID(gomock.Any(), storyID).Return(nil, audio.ErrNotFound)
				stg.EXPECT().Get(gomock.Any(), testUserID).Return(nil, fmt.Errorf("settings db error"))
				el.EXPECT().GenerateSpeech(gomock.Any(), "test-voice", "テスト").Return([]byte("audio"), nil)
				fs.EXPECT().Write(storyID.String()+".mp3", []byte("audio")).Return(nil)
				ar.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
			check: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "audio/mpeg", w.Header().Get("Content-Type"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ar := audiomock.NewMockRepository(ctrl)
			fs := audiomock.NewMockFileStore(ctrl)
			sr := storymock.NewMockRepository(ctrl)
			el := elevenlabsmock.NewMockClient(ctrl)
			stg := domainmock.NewMockSettingsRepository(ctrl)
			tt.setup(ar, fs, sr, el, stg)

			sm := testSessionManager(t)
			srv := NewServer(
				context.Background(),
				logr.Discard(),
				sm, nil, nil, "shiru_session", 72*time.Hour, false, "http://localhost:5173",
				stg,
				domainmock.NewMockVocabRepository(ctrl),
				sr, nil, nil, nil, nil, nil, el, nil, ar, fs, nil, testVoiceSelector,
			)
			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			addAuthCookie(t, sm, req)
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.check != nil {
				tt.check(t, w)
			}
		})
	}
}

func TestCreateStoryAudioMetadata(t *testing.T) {
	t.Parallel()

	storyID := uuid.New()
	ctrl := gomock.NewController(t)
	ar := audiomock.NewMockRepository(ctrl)
	fs := audiomock.NewMockFileStore(ctrl)
	sr := storymock.NewMockRepository(ctrl)
	el := elevenlabsmock.NewMockClient(ctrl)
	stg := domainmock.NewMockSettingsRepository(ctrl)

	vs := elevenlabs.NewVoiceSelector("fallback-voice", map[string]string{
		"N4_funny_male":   "n4-funny-m",
		"N4_funny_female": "n4-funny-f",
	})

	sr.EXPECT().Get(gomock.Any(), testUserID, storyID).Return(&story.Story{
		ID:      storyID,
		Content: "テスト",
		Tone:    "funny",
	}, nil)
	stg.EXPECT().Get(gomock.Any(), testUserID).Return(&domain.UserSettings{JLPTLevel: "N4"}, nil)
	ar.EXPECT().GetByStoryID(gomock.Any(), storyID).Return(nil, audio.ErrNotFound)
	el.EXPECT().GenerateSpeech(gomock.Any(), gomock.Any(), "テスト").Return([]byte("audio"), nil)
	fs.EXPECT().Write(gomock.Any(), gomock.Any()).Return(nil)
	ar.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ interface{}, sa *audio.StoryAudio) error {
		assert.Equal(t, storyID, sa.StoryID)
		assert.Contains(t, []string{"n4-funny-m", "n4-funny-f"}, sa.VoiceID)
		assert.Equal(t, "mp3", sa.AudioFormat)
		assert.Equal(t, storyID.String()+".mp3", sa.StoragePath)
		require.NotEmpty(t, sa.Checksum)
		return nil
	})

	sm := testSessionManager(t)
	srv := NewServer(
		context.Background(),
		logr.Discard(),
		sm, nil, nil, "shiru_session", 72*time.Hour, false, "http://localhost:5173",
		stg,
		domainmock.NewMockVocabRepository(ctrl),
		sr, nil, nil, nil, nil, nil, el, nil, ar, fs, nil, vs,
	)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/stories/%s/audio", storyID), nil)
	addAuthCookie(t, sm, req)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}
