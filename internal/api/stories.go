package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/0x5d/shiru/internal/domain"
	"github.com/0x5d/shiru/internal/story"
	"github.com/google/uuid"
)

type searchResultResponse struct {
	StoryID   uuid.UUID `json:"story_id"`
	Topic     string    `json:"topic"`
	Tone      string    `json:"tone"`
	Content   string    `json:"content"`
	JLPTLevel string    `json:"jlpt_level"`
	CreatedAt time.Time `json:"created_at"`
}

type searchStoriesResponse struct {
	Results []searchResultResponse `json:"results"`
	Total   int                    `json:"total"`
}

func (s *Server) searchStories(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "q parameter required", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	results, total, err := s.es.SearchStories(r.Context(), domain.DefaultUserID.String(), q, limit, offset)
	if err != nil {
		s.log.Error(err, "failed to search stories")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := searchStoriesResponse{
		Results: make([]searchResultResponse, len(results)),
		Total:   total,
	}
	for i, r := range results {
		resp.Results[i] = searchResultResponse{
			StoryID:   r.StoryID,
			Topic:     r.Topic,
			Tone:      r.Tone,
			Content:   r.Content,
			JLPTLevel: r.JLPTLevel,
			CreatedAt: r.CreatedAt,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

type tokenResponse struct {
	Surface      string     `json:"surface"`
	StartOffset  int        `json:"start_offset"`
	EndOffset    int        `json:"end_offset"`
	VocabEntryID *uuid.UUID `json:"vocab_entry_id,omitempty"`
	IsVocabMatch bool       `json:"is_vocab_match"`
}

type storyTokensResponse struct {
	StoryID uuid.UUID       `json:"story_id"`
	Tokens  []tokenResponse `json:"tokens"`
}

func (s *Server) getStoryTokens(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(r.PathValue("storyID"))
	if err != nil {
		http.Error(w, "invalid story ID", http.StatusBadRequest)
		return
	}

	st, err := s.storyRepo.Get(r.Context(), storyID)
	if err != nil {
		if errors.Is(err, story.ErrNotFound) {
			http.Error(w, "story not found", http.StatusNotFound)
			return
		}
		s.log.Error(err, "failed to get story")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tokens, err := s.es.Tokenize(r.Context(), st.Content)
	if err != nil {
		s.log.Error(err, "failed to tokenize story")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	surfaces := make([]string, len(tokens))
	for i, t := range tokens {
		surfaces[i] = domain.NormalizeSurface(t.Surface)
	}

	vocabEntries, err := s.vocab.GetByNormalizedSurfaces(r.Context(), st.UserID, surfaces)
	if err != nil {
		s.log.Error(err, "failed to match vocab")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	vocabMap := make(map[string]uuid.UUID, len(vocabEntries))
	for _, v := range vocabEntries {
		vocabMap[v.NormalizedSurface] = v.ID
	}

	respTokens := make([]tokenResponse, len(tokens))
	for i, t := range tokens {
		tr := tokenResponse{
			Surface:     t.Surface,
			StartOffset: t.StartOffset,
			EndOffset:   t.EndOffset,
		}
		normalized := domain.NormalizeSurface(t.Surface)
		if id, ok := vocabMap[normalized]; ok {
			tr.VocabEntryID = &id
			tr.IsVocabMatch = true
		}
		respTokens[i] = tr
	}

	writeJSON(w, http.StatusOK, storyTokensResponse{
		StoryID: storyID,
		Tokens:  respTokens,
	})
}
