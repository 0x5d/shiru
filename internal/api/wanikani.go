package api

import (
	"net/http"
	"time"

	"github.com/0x5d/shiru/internal/domain"
)

type importWaniKaniResponse struct {
	ImportedCount int `json:"imported_count"`
}

func (s *Server) importWaniKani(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settings.Get(r.Context(), domain.DefaultUserID)
	if err != nil {
		s.log.Error(err, "failed to get settings")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if settings.WaniKaniAPIKey == nil || *settings.WaniKaniAPIKey == "" {
		http.Error(w, "WaniKani API key not configured", http.StatusBadRequest)
		return
	}

	items, err := s.wanikani.FetchVocabulary(r.Context(), *settings.WaniKaniAPIKey, settings.WaniKaniLastSyncedAt)
	if err != nil {
		s.log.Error(err, "failed to fetch WaniKani vocabulary")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	syncTime := time.Now().UTC()

	if len(items) > 0 {
		surfaces := make([]string, len(items))
		for i, item := range items {
			surfaces[i] = item.Characters
		}

		entries, err := s.vocab.BatchUpsert(r.Context(), domain.DefaultUserID, surfaces, "wanikani")
		if err != nil {
			s.log.Error(err, "failed to upsert WaniKani vocab")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		s.generateTagsForEntries(r.Context(), domain.DefaultUserID, entries)
	}

	if err := s.settings.UpdateWaniKaniSyncedAt(r.Context(), domain.DefaultUserID, syncTime); err != nil {
		s.log.Error(err, "failed to update wanikani synced at")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, importWaniKaniResponse{
		ImportedCount: len(items),
	})
}
