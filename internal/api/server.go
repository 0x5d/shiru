package api

import (
	"net/http"

	"github.com/0x5d/shiru/internal/dictionary"
	"github.com/0x5d/shiru/internal/domain"
	"github.com/0x5d/shiru/internal/elasticsearch"
	"github.com/0x5d/shiru/internal/story"
	"github.com/go-logr/logr"
)

type Server struct {
	settings   domain.SettingsRepository
	vocab      domain.VocabRepository
	storyRepo  story.Repository
	es         elasticsearch.Client
	dictionary dictionary.Client
	log        logr.Logger
	mux        *http.ServeMux
}

func NewServer(
	log logr.Logger,
	settings domain.SettingsRepository,
	vocab domain.VocabRepository,
	storyRepo story.Repository,
	es elasticsearch.Client,
	dictionary dictionary.Client,
) *Server {
	s := &Server{
		settings:   settings,
		vocab:      vocab,
		storyRepo:  storyRepo,
		es:         es,
		dictionary: dictionary,
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
	s.mux.HandleFunc("GET /api/v1/stories/search", s.searchStories)
	s.mux.HandleFunc("GET /api/v1/stories/{storyID}/tokens", s.getStoryTokens)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
