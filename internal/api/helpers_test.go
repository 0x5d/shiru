package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0x5d/shiru/internal/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testUserID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

func addAuthCookie(t *testing.T, sm *auth.SessionManager, req *http.Request) {
	t.Helper()
	token, err := sm.Encode(testUserID)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "shiru_session", Value: token})
}

func assertNoSessionCookie(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		assert.NotEqual(t, "shiru_session", c.Name,
			"no session cookie should be set on error responses")
	}
}
