package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/0x5d/shiru/internal/domain"
)

const maxAuthBodySize = 8 * 1024

type googleLoginRequest struct {
	Credential string `json:"credential"`
}

type userResponse struct {
	ID        string  `json:"id"`
	Email     *string `json:"email"`
	Name      *string `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

func (s *Server) googleLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodySize)
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Credential == "" {
		http.Error(w, "missing credential", http.StatusBadRequest)
		return
	}

	claims, err := s.googleVerifier.Verify(r.Context(), req.Credential)
	if err != nil {
		s.log.Error(err, "google token verification failed")
		http.Error(w, "invalid credential", http.StatusUnauthorized)
		return
	}

	user, err := s.users.UpsertGoogleUser(r.Context(), claims.Sub, claims.Email, claims.Name, claims.AvatarURL)
	if err != nil {
		s.log.Error(err, "upserting google user")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.users.EnsureUserSettings(r.Context(), user.ID); err != nil {
		s.log.Error(err, "ensuring user settings")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := s.sessions.Encode(user.ID)
	if err != nil {
		s.log.Error(err, "encoding session")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{ // nosemgrep: go.lang.security.audit.net.cookie-missing-secure.cookie-missing-secure
		Name:     s.cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(s.sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, userResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
	})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(s.cookieName)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	payload, err := s.sessions.Decode(cookie.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.users.GetByID(r.Context(), payload.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.log.Error(err, "getting user by id")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{ // nosemgrep: go.lang.security.audit.net.cookie-missing-secure.cookie-missing-secure
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusNoContent)
}
