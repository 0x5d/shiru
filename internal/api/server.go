package api

import (
	"context"
	"net/http"

	shiruanthropic "github.com/0x5d/shiru/internal/anthropic"
	"github.com/0x5d/shiru/internal/audio"
	"github.com/0x5d/shiru/internal/dictionary"
	"github.com/0x5d/shiru/internal/domain"
	"github.com/0x5d/shiru/internal/elasticsearch"
	"github.com/0x5d/shiru/internal/elevenlabs"
	"github.com/0x5d/shiru/internal/story"
	"github.com/0x5d/shiru/internal/wanikani"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

type TagRepository interface {
	ListUserTags(ctx context.Context, userID uuid.UUID) ([]string, error)
	UpsertTagsAndLink(ctx context.Context, userID uuid.UUID, vocabEntryID uuid.UUID, tagNames []string) error
}

type Server struct {
	settings     domain.SettingsRepository
	vocab        domain.VocabRepository
	storyRepo    story.Repository
	storySvc     *story.Service
	tags         TagRepository
	anthropic    shiruanthropic.Client
	es           elasticsearch.Client
	dictionary   dictionary.Client
	elevenlabs   elevenlabs.Client
	wanikani     wanikani.Client
	audioRepo    audio.Repository
	audioStore   audio.FileStore
	voiceID      string
	log          logr.Logger
	mux          *http.ServeMux
}

func NewServer(
	log logr.Logger,
	settings domain.SettingsRepository,
	vocab domain.VocabRepository,
	storyRepo story.Repository,
	storySvc *story.Service,
	tags TagRepository,
	anthropic shiruanthropic.Client,
	es elasticsearch.Client,
	dictionary dictionary.Client,
	el elevenlabs.Client,
	wk wanikani.Client,
	audioRepo audio.Repository,
	audioStore audio.FileStore,
	voiceID string,
) *Server {
	s := &Server{
		settings:   settings,
		vocab:      vocab,
		storyRepo:  storyRepo,
		storySvc:   storySvc,
		tags:       tags,
		anthropic:  anthropic,
		es:         es,
		dictionary: dictionary,
		elevenlabs: el,
		wanikani:   wk,
		audioRepo:  audioRepo,
		audioStore: audioStore,
		voiceID:    voiceID,
		log:        log,
		mux:        http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/settings", s.getSettings)
	s.mux.HandleFunc("PUT /api/v1/settings", s.updateSettings)
	s.mux.HandleFunc("GET /api/v1/vocab", s.listVocab)
	s.mux.HandleFunc("POST /api/v1/vocab", s.createVocab)
	s.mux.HandleFunc("GET /api/v1/vocab/{vocabID}/details", s.getVocabDetails)
	s.mux.HandleFunc("GET /api/v1/dictionary/lookup", s.lookupWord)
	s.mux.HandleFunc("POST /api/v1/topics/generate", s.generateTopics)
	s.mux.HandleFunc("POST /api/v1/stories", s.createStory)
	s.mux.HandleFunc("GET /api/v1/stories/search", s.searchStories)
	s.mux.HandleFunc("GET /api/v1/stories/{storyID}/tokens", s.getStoryTokens)
	s.mux.HandleFunc("GET /api/v1/stories/{storyID}", s.getStory)
	s.mux.HandleFunc("GET /api/v1/stories", s.listStories)
	s.mux.HandleFunc("POST /api/v1/stories/{storyID}/audio", s.createStoryAudio)
	s.mux.HandleFunc("POST /api/v1/vocab/import/wanikani", s.importWaniKani)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}
