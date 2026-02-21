package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/0x5d/shiru/internal/domain"
)

type settingsResponse struct {
	JLPTLevel            string     `json:"jlpt_level"`
	StoryWordTarget      int        `json:"story_word_target"`
	WaniKaniAPIKey       *string    `json:"wanikani_api_key,omitempty"`
	WaniKaniLastSyncedAt *time.Time `json:"wanikani_last_synced_at,omitempty"`
}

type updateSettingsRequest struct {
	JLPTLevel       string  `json:"jlpt_level"`
	StoryWordTarget int     `json:"story_word_target"`
	WaniKaniAPIKey  *string `json:"wanikani_api_key"`
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settings.Get(r.Context(), domain.DefaultUserID)
	if err != nil {
		s.log.Error(err, "failed to get settings")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, settingsResponse{
		JLPTLevel:            settings.JLPTLevel,
		StoryWordTarget:      settings.StoryWordTarget,
		WaniKaniAPIKey:       settings.WaniKaniAPIKey,
		WaniKaniLastSyncedAt: settings.WaniKaniLastSyncedAt,
	})
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !validJLPTLevel(req.JLPTLevel) {
		http.Error(w, "invalid jlpt_level", http.StatusBadRequest)
		return
	}
	if req.StoryWordTarget < 50 || req.StoryWordTarget > 500 {
		http.Error(w, "story_word_target must be between 50 and 500", http.StatusBadRequest)
		return
	}

	settings, err := s.settings.Update(r.Context(), domain.DefaultUserID, req.JLPTLevel, req.StoryWordTarget, req.WaniKaniAPIKey)
	if err != nil {
		s.log.Error(err, "failed to update settings")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, settingsResponse{
		JLPTLevel:            settings.JLPTLevel,
		StoryWordTarget:      settings.StoryWordTarget,
		WaniKaniAPIKey:       settings.WaniKaniAPIKey,
		WaniKaniLastSyncedAt: settings.WaniKaniLastSyncedAt,
	})
}

func validJLPTLevel(level string) bool {
	switch level {
	case "N5", "N4", "N3", "N2", "N1":
		return true
	}
	return false
}
