package api

import (
	"net/http"
	"testing"

	"github.com/0x5d/shiru/internal/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var testUserID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

func addAuthCookie(t *testing.T, sm *auth.SessionManager, req *http.Request) {
	t.Helper()
	token, err := sm.Encode(testUserID)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "shiru_session", Value: token})
}
