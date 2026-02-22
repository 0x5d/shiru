package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	shiruanthropic "github.com/0x5d/shiru/internal/anthropic"
	"github.com/0x5d/shiru/internal/audio"
	"github.com/0x5d/shiru/internal/auth"
	"github.com/0x5d/shiru/internal/dictionary"
	"github.com/0x5d/shiru/internal/domain"
	"github.com/0x5d/shiru/internal/elasticsearch"
	"github.com/0x5d/shiru/internal/elevenlabs"
	"github.com/0x5d/shiru/internal/postgres"
	"github.com/0x5d/shiru/internal/story"
	"github.com/0x5d/shiru/internal/wanikani"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

type GoogleTokenVerifier interface {
	Verify(ctx context.Context, credential string) (*auth.GoogleClaims, error)
}

type TagRepository interface {
	ListUserTags(ctx context.Context, userID uuid.UUID) ([]string, error)
	UpsertTagsAndLink(ctx context.Context, userID uuid.UUID, vocabEntryID uuid.UUID, tagNames []string) error
}

type Server struct {
	sessions       *auth.SessionManager
	googleVerifier GoogleTokenVerifier
	users          domain.UserRepository
	cookieName     string
	sessionTTL     time.Duration
	secureCookies  bool
	allowedOrigin  string
	settings       domain.SettingsRepository
	vocab          domain.VocabRepository
	storyRepo      story.Repository
	storySvc       *story.Service
	tags           TagRepository
	anthropic      shiruanthropic.Client
	es             elasticsearch.Client
	dictionary     dictionary.Client
	elevenlabs     elevenlabs.Client
	wanikani       wanikani.Client
	audioRepo      audio.Repository
	audioStore     audio.FileStore
	topicSnapshots *postgres.TopicSnapshotRepository
	voiceID        string
	log            logr.Logger
	mux            *http.ServeMux
	bgCtx          context.Context
	bgWg           sync.WaitGroup
}

func NewServer(
	bgCtx context.Context,
	log logr.Logger,
	sessions *auth.SessionManager,
	googleVerifier GoogleTokenVerifier,
	users domain.UserRepository,
	cookieName string,
	sessionTTL time.Duration,
	secureCookies bool,
	allowedOrigin string,
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
	topicSnapshots *postgres.TopicSnapshotRepository,
	voiceID string,
) *Server {
	s := &Server{
		sessions:       sessions,
		googleVerifier: googleVerifier,
		users:          users,
		cookieName:     cookieName,
		sessionTTL:     sessionTTL,
		secureCookies:  secureCookies,
		allowedOrigin:  allowedOrigin,
		settings:       settings,
		vocab:          vocab,
		storyRepo:      storyRepo,
		storySvc:       storySvc,
		tags:           tags,
		anthropic:      anthropic,
		es:             es,
		dictionary:     dictionary,
		elevenlabs:     el,
		wanikani:       wk,
		audioRepo:      audioRepo,
		audioStore:     audioStore,
		topicSnapshots: topicSnapshots,
		voiceID:        voiceID,
		log:            log,
		mux:            http.NewServeMux(),
		bgCtx:          bgCtx,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/v1/auth/google", s.googleLogin)
	s.mux.HandleFunc("GET /api/v1/auth/me", s.me)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.logout)

	s.mux.HandleFunc("GET /api/v1/settings", s.requireAuth(s.getSettings))
	s.mux.HandleFunc("PUT /api/v1/settings", s.requireAuth(s.updateSettings))
	s.mux.HandleFunc("GET /api/v1/vocab", s.requireAuth(s.listVocab))
	s.mux.HandleFunc("POST /api/v1/vocab", s.requireAuth(s.createVocab))
	s.mux.HandleFunc("GET /api/v1/vocab/{vocabID}/details", s.requireAuth(s.getVocabDetails))
	s.mux.HandleFunc("GET /api/v1/dictionary/lookup", s.requireAuth(s.lookupWord))
	s.mux.HandleFunc("GET /api/v1/topics", s.requireAuth(s.generateTopics))
	s.mux.HandleFunc("POST /api/v1/stories", s.requireAuth(s.createStory))
	s.mux.HandleFunc("GET /api/v1/stories/search", s.requireAuth(s.searchStories))
	s.mux.HandleFunc("GET /api/v1/stories/{storyID}/tokens", s.requireAuth(s.getStoryTokens))
	s.mux.HandleFunc("GET /api/v1/stories/{storyID}", s.requireAuth(s.getStory))
	s.mux.HandleFunc("GET /api/v1/stories", s.requireAuth(s.listStories))
	s.mux.HandleFunc("POST /api/v1/stories/{storyID}/audio", s.requireAuth(s.createStoryAudio))
	s.mux.HandleFunc("POST /api/v1/vocab/import/wanikani", s.requireAuth(s.importWaniKani))
}

func (s *Server) WaitForBackground() {
	s.bgWg.Wait()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Vary", "Origin")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}
