package api

import (
	"net/http"

	"github.com/google/uuid"
)

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		if payload.UserID == uuid.Nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := contextWithUserID(r.Context(), payload.UserID)
		next(w, r.WithContext(ctx))
	}
}
