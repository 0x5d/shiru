package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/0x5d/shiru/internal/domain"
	"github.com/google/uuid"
)

type vocabEntryResponse struct {
	ID                uuid.UUID `json:"id"`
	Surface           string    `json:"surface"`
	NormalizedSurface string    `json:"normalized_surface"`
	Meaning           *string   `json:"meaning,omitempty"`
	Reading           *string   `json:"reading,omitempty"`
	Source            string    `json:"source"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type listVocabResponse struct {
	Entries []vocabEntryResponse `json:"entries"`
	Total   int                  `json:"total"`
}

type createVocabRequest struct {
	Entries []string `json:"entries"`
}

type createVocabResponse struct {
	Entries []vocabEntryResponse `json:"entries"`
}

type vocabStatusResponse struct {
	TotalVocab        int  `json:"total_vocab"`
	TaggedVocabCount  int  `json:"tagged_vocab_count"`
	TaggingInProgress bool `json:"tagging_in_progress"`
}

func (s *Server) getVocabStatus(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())

	_, totalVocab, err := s.vocab.List(r.Context(), userID, "", 0, 0)
	if err != nil {
		s.log.Error(err, "failed to count vocab")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	taggedCount, err := s.tags.CountTaggedVocab(r.Context(), userID)
	if err != nil {
		s.log.Error(err, "failed to count tagged vocab")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_, tagging := s.taggingUsers.Load(userID)

	writeJSON(w, http.StatusOK, vocabStatusResponse{
		TotalVocab:        totalVocab,
		TaggedVocabCount:  taggedCount,
		TaggingInProgress: tagging,
	})
}

func (s *Server) listVocab(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	userID := userIDFromContext(r.Context())
	entries, total, err := s.vocab.List(r.Context(), userID, query, limit, offset)
	if err != nil {
		s.log.Error(err, "failed to list vocab")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := listVocabResponse{
		Entries: make([]vocabEntryResponse, len(entries)),
		Total:   total,
	}
	for i, e := range entries {
		resp.Entries[i] = toVocabEntryResponse(e)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) createVocab(w http.ResponseWriter, r *http.Request) {
	var req createVocabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Entries) == 0 {
		http.Error(w, "entries required", http.StatusBadRequest)
		return
	}

	userID := userIDFromContext(r.Context())
	entries, err := s.vocab.BatchUpsert(r.Context(), userID, req.Entries, "manual")
	if err != nil {
		s.log.Error(err, "failed to create vocab")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.generateTagsForEntries(r.Context(), userID, entries)

	resp := createVocabResponse{
		Entries: make([]vocabEntryResponse, len(entries)),
	}
	for i, e := range entries {
		resp.Entries[i] = toVocabEntryResponse(e)
	}
	writeJSON(w, http.StatusCreated, resp)
}

type vocabDetailsResponse struct {
	ID      uuid.UUID `json:"id"`
	Surface string    `json:"surface"`
	Meaning string    `json:"meaning"`
	Reading string    `json:"reading"`
}

func (s *Server) getVocabDetails(w http.ResponseWriter, r *http.Request) {
	vocabID, err := uuid.Parse(r.PathValue("vocabID"))
	if err != nil {
		http.Error(w, "invalid vocab ID", http.StatusBadRequest)
		return
	}

	userID := userIDFromContext(r.Context())
	entry, err := s.vocab.GetByID(r.Context(), userID, vocabID)
	if err != nil {
		if errors.Is(err, domain.ErrVocabNotFound) {
			http.Error(w, "vocab entry not found", http.StatusNotFound)
			return
		}
		s.log.Error(err, "failed to get vocab entry")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	meaning := ptrVal(entry.Meaning)
	reading := ptrVal(entry.Reading)

	if meaning == "" || reading == "" {
		result, err := s.dictionary.Lookup(r.Context(), entry.Surface)
		if err != nil {
			s.log.Error(err, "dictionary lookup failed", "surface", entry.Surface)
		} else {
			if meaning == "" {
				meaning = result.Meaning
			}
			if reading == "" {
				reading = result.Reading
			}
			if err := s.vocab.UpdateDetails(r.Context(), userID, entry.ID, meaning, reading); err != nil {
				s.log.Error(err, "failed to cache vocab details")
			}
		}
	}

	writeJSON(w, http.StatusOK, vocabDetailsResponse{
		ID:      entry.ID,
		Surface: entry.Surface,
		Meaning: meaning,
		Reading: reading,
	})
}

type lookupWordResponse struct {
	Meaning string `json:"meaning"`
	Reading string `json:"reading"`
}

func (s *Server) lookupWord(w http.ResponseWriter, r *http.Request) {
	word := r.URL.Query().Get("word")
	if word == "" {
		http.Error(w, "missing word parameter", http.StatusBadRequest)
		return
	}

	result, err := s.dictionary.Lookup(r.Context(), word)
	if err != nil {
		s.log.Error(err, "dictionary lookup failed", "word", word)
		http.Error(w, "word not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, lookupWordResponse{
		Meaning: result.Meaning,
		Reading: result.Reading,
	})
}

func (s *Server) generateTagsForEntries(ctx context.Context, userID uuid.UUID, entries []domain.VocabEntry) {
	if s.anthropic == nil || len(entries) == 0 {
		return
	}

	surfaces := make([]string, len(entries))
	byName := make(map[string]uuid.UUID, len(entries))
	for i, entry := range entries {
		surfaces[i] = entry.Surface
		byName[entry.Surface] = entry.ID
	}

	tagMap, err := s.anthropic.GenerateTagsBatch(ctx, surfaces)
	if err != nil {
		s.log.Error(err, "failed to generate tags batch")
		return
	}

	for surface, tags := range tagMap {
		entryID, ok := byName[surface]
		if !ok {
			continue
		}
		if err := s.tags.UpsertTagsAndLink(ctx, userID, entryID, tags); err != nil {
			s.log.Error(err, "failed to store tags", "surface", surface)
		}
	}
}

func (s *Server) deleteAllVocab(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if err := s.vocab.DeleteAll(r.Context(), userID); err != nil {
		s.log.Error(err, "failed to delete all vocab")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.settings.ResetWaniKaniSyncedAt(r.Context(), userID); err != nil {
		s.log.Error(err, "failed to reset wanikani sync timestamp")
	}
	w.WriteHeader(http.StatusNoContent)
}

func ptrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toVocabEntryResponse(e domain.VocabEntry) vocabEntryResponse {
	return vocabEntryResponse{
		ID:                e.ID,
		Surface:           e.Surface,
		NormalizedSurface: e.NormalizedSurface,
		Meaning:           e.Meaning,
		Reading:           e.Reading,
		Source:            e.Source,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}
