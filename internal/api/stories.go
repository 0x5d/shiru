package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/0x5d/shiru/internal/domain"
	"github.com/0x5d/shiru/internal/story"
	"github.com/google/uuid"
)

type generateTopicsResponse struct {
	Topics []string `json:"topics"`
}

func (s *Server) generateTopics(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settings.Get(r.Context(), domain.DefaultUserID)
	if err != nil {
		s.log.Error(err, "failed to get settings")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tags, err := s.tags.ListUserTags(r.Context(), domain.DefaultUserID)
	if err != nil {
		s.log.Error(err, "failed to list user tags")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	topics, err := s.storySvc.GenerateTopics(r.Context(), tags, settings.JLPTLevel)
	if err != nil {
		s.log.Error(err, "failed to generate topics")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, generateTopicsResponse{Topics: topics})
}

type createStoryRequest struct {
	Topic string `json:"topic"`
}

type storyResponse struct {
	ID              uuid.UUID `json:"id"`
	Topic           string    `json:"topic"`
	Title           string    `json:"title"`
	Tone            string    `json:"tone"`
	JLPTLevel       string    `json:"jlpt_level"`
	TargetWordCount int       `json:"target_word_count"`
	ActualWordCount int       `json:"actual_word_count"`
	Content         string    `json:"content"`
	UsedVocabCount  int       `json:"used_vocab_count"`
	SourceTagNames  []string  `json:"source_tag_names"`
	CreatedAt       time.Time `json:"created_at"`
}

func toStoryResponse(st *story.Story) storyResponse {
	return storyResponse{
		ID:              st.ID,
		Topic:           st.Topic,
		Title:           st.Title,
		Tone:            st.Tone,
		JLPTLevel:       st.JLPTLevel,
		TargetWordCount: st.TargetWordCount,
		ActualWordCount: st.ActualWordCount,
		Content:         st.Content,
		UsedVocabCount:  st.UsedVocabCount,
		SourceTagNames:  st.SourceTagNames,
		CreatedAt:       st.CreatedAt,
	}
}

func (s *Server) createStory(w http.ResponseWriter, r *http.Request) {
	var req createStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Topic == "" {
		http.Error(w, "topic required", http.StatusBadRequest)
		return
	}

	settings, err := s.settings.Get(r.Context(), domain.DefaultUserID)
	if err != nil {
		s.log.Error(err, "failed to get settings")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tags, err := s.tags.ListUserTags(r.Context(), domain.DefaultUserID)
	if err != nil {
		s.log.Error(err, "failed to list user tags")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	st, err := s.storySvc.Generate(r.Context(), story.GenerateParams{
		UserID:          domain.DefaultUserID,
		Topic:           req.Topic,
		Tags:            tags,
		JLPTLevel:       settings.JLPTLevel,
		TargetWordCount: settings.StoryWordTarget,
	})
	if err != nil {
		s.log.Error(err, "failed to generate story")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, toStoryResponse(st))
}

func (s *Server) getStory(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, toStoryResponse(st))
}

type listStoriesResponse struct {
	Stories []storyResponse `json:"stories"`
}

func (s *Server) listStories(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	stories, err := s.storyRepo.List(r.Context(), domain.DefaultUserID, limit, offset)
	if err != nil {
		s.log.Error(err, "failed to list stories")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := listStoriesResponse{
		Stories: make([]storyResponse, len(stories)),
	}
	for i, st := range stories {
		resp.Stories[i] = toStoryResponse(st)
	}
	writeJSON(w, http.StatusOK, resp)
}

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
	Reading      string     `json:"reading,omitempty"`
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
			Reading:     t.Reading,
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
