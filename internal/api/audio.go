package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"

	"github.com/0x5d/shiru/internal/audio"
	"github.com/0x5d/shiru/internal/story"
	"github.com/google/uuid"
)

func (s *Server) createStoryAudio(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(r.PathValue("storyID"))
	if err != nil {
		http.Error(w, "invalid story ID", http.StatusBadRequest)
		return
	}

	existing, err := s.audioRepo.GetByStoryID(r.Context(), storyID)
	if err != nil && !errors.Is(err, audio.ErrNotFound) {
		s.log.Error(err, "failed to check cached audio")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if existing != nil {
		data, err := s.audioStore.Read(existing.StoragePath)
		if err != nil {
			s.log.Error(err, "failed to read cached audio file", "path", existing.StoragePath)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data) // nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
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

	audioData, err := s.elevenlabs.GenerateSpeech(r.Context(), st.Content)
	if err != nil {
		s.log.Error(err, "failed to generate speech")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(audioData))
	storagePath := storyID.String() + ".mp3"

	if err := s.audioStore.Write(storagePath, audioData); err != nil {
		s.log.Error(err, "failed to write audio file")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	storyAudio := &audio.StoryAudio{
		StoryID:     storyID,
		VoiceID:     s.voiceID,
		AudioFormat: "mp3",
		StoragePath: storagePath,
		Checksum:    checksum,
	}
	if err := s.audioRepo.Create(r.Context(), storyAudio); err != nil {
		s.log.Error(err, "failed to save audio metadata")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(audioData) // nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
}
