package wanikani

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchVocabulary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		updatedAfter *time.Time
		handler      http.HandlerFunc
		wantItems    []VocabItem
		wantErr      string
	}{
		{
			name: "fetches vocabulary successfully",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				assert.Equal(t, "20170710", r.Header.Get("Wanikani-Revision"))

				switch r.URL.Path {
				case "/assignments":
					assert.Equal(t, "vocabulary,kana_vocabulary", r.URL.Query().Get("subject_types"))
					assert.Equal(t, "true", r.URL.Query().Get("unlocked"))
					writeTestJSON(t, w, assignmentsResponse{
						Data: []assignmentItem{
							{Data: assignmentData{SubjectID: 100}},
							{Data: assignmentData{SubjectID: 200}},
						},
					})
				case "/subjects":
					assert.Equal(t, "vocabulary,kana_vocabulary", r.URL.Query().Get("types"))
					writeTestJSON(t, w, subjectsResponse{
						Data: []subjectItem{
							{ID: 100, Data: subjectData{Characters: "花"}},
							{ID: 200, Data: subjectData{Characters: "走る"}},
						},
					})
				}
			}),
			wantItems: []VocabItem{
				{SubjectID: 100, Characters: "花"},
				{SubjectID: 200, Characters: "走る"},
			},
		},
		{
			name: "incremental sync with updated_after",
			updatedAfter: func() *time.Time {
				t := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				return &t
			}(),
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/assignments":
					assert.Equal(t, "2026-01-01T00:00:00Z", r.URL.Query().Get("updated_after"))
					writeTestJSON(t, w, assignmentsResponse{
						Data: []assignmentItem{
							{Data: assignmentData{SubjectID: 300}},
						},
					})
				case "/subjects":
					writeTestJSON(t, w, subjectsResponse{
						Data: []subjectItem{
							{ID: 300, Data: subjectData{Characters: "猫"}},
						},
					})
				}
			}),
			wantItems: []VocabItem{
				{SubjectID: 300, Characters: "猫"},
			},
		},
		{
			name: "no assignments returns nil",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/assignments" {
					writeTestJSON(t, w, assignmentsResponse{})
				}
			}),
			wantItems: nil,
		},
		{
			name: "handles pagination",
			handler: func() http.HandlerFunc {
				assignmentCalls := 0
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/assignments":
						assignmentCalls++
						if assignmentCalls == 1 {
							writeTestJSON(t, w, assignmentsResponse{
								Pages: paginationPages{NextURL: "http://" + r.Host + "/assignments?page=2&subject_types=vocabulary&unlocked=true"},
								Data: []assignmentItem{
									{Data: assignmentData{SubjectID: 100}},
								},
							})
						} else {
							writeTestJSON(t, w, assignmentsResponse{
								Data: []assignmentItem{
									{Data: assignmentData{SubjectID: 200}},
								},
							})
						}
					case "/subjects":
						writeTestJSON(t, w, subjectsResponse{
							Data: []subjectItem{
								{ID: 100, Data: subjectData{Characters: "花"}},
								{ID: 200, Data: subjectData{Characters: "猫"}},
							},
						})
					}
				}
			}(),
			wantItems: []VocabItem{
				{SubjectID: 100, Characters: "花"},
				{SubjectID: 200, Characters: "猫"},
			},
		},
		{
			name: "unauthorized error",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			}),
			wantErr: "invalid API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := New(server.URL)
			items, err := client.FetchVocabulary(context.Background(), "test-key", tt.updatedAfter)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantItems, items)
		})
	}
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(v))
}
