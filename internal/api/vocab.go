package api

import (
	"encoding/json"
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

	entries, total, err := s.vocab.List(r.Context(), domain.DefaultUserID, query, limit, offset)
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

	entries, err := s.vocab.BatchUpsert(r.Context(), domain.DefaultUserID, req.Entries, "manual")
	if err != nil {
		s.log.Error(err, "failed to create vocab")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := createVocabResponse{
		Entries: make([]vocabEntryResponse, len(entries)),
	}
	for i, e := range entries {
		resp.Entries[i] = toVocabEntryResponse(e)
	}
	writeJSON(w, http.StatusCreated, resp)
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
