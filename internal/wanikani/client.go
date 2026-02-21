package wanikani

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:generate go run go.uber.org/mock/mockgen -destination mock/client.go -package mock . Client

type Client interface {
	FetchVocabulary(ctx context.Context, apiKey string, updatedAfter *time.Time) ([]VocabItem, error)
}

type VocabItem struct {
	SubjectID  int
	Characters string
}

var _ Client = (*wanikaniClient)(nil)

type wanikaniClient struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *wanikaniClient {
	return &wanikaniClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *wanikaniClient) FetchVocabulary(ctx context.Context, apiKey string, updatedAfter *time.Time) ([]VocabItem, error) {
	subjectIDs, err := c.fetchAssignmentSubjectIDs(ctx, apiKey, updatedAfter)
	if err != nil {
		return nil, fmt.Errorf("fetching assignments: %w", err)
	}
	if len(subjectIDs) == 0 {
		return nil, nil
	}

	subjects, err := c.fetchSubjects(ctx, apiKey, subjectIDs)
	if err != nil {
		return nil, fmt.Errorf("fetching subjects: %w", err)
	}

	return subjects, nil
}

func (c *wanikaniClient) fetchAssignmentSubjectIDs(ctx context.Context, apiKey string, updatedAfter *time.Time) ([]int, error) {
	url := c.baseURL + "/assignments?subject_types=vocabulary&unlocked=true"
	if updatedAfter != nil {
		url += "&updated_after=" + updatedAfter.UTC().Format(time.RFC3339)
	}

	var subjectIDs []int
	for url != "" {
		var resp assignmentsResponse
		if err := c.doGet(ctx, apiKey, url, &resp); err != nil {
			return nil, err
		}
		for _, item := range resp.Data {
			subjectIDs = append(subjectIDs, item.Data.SubjectID)
		}
		url = resp.Pages.NextURL
	}

	return subjectIDs, nil
}

func (c *wanikaniClient) fetchSubjects(ctx context.Context, apiKey string, subjectIDs []int) ([]VocabItem, error) {
	var items []VocabItem

	const batchSize = 1000
	for i := 0; i < len(subjectIDs); i += batchSize {
		end := i + batchSize
		if end > len(subjectIDs) {
			end = len(subjectIDs)
		}
		batch := subjectIDs[i:end]

		idStrs := make([]string, len(batch))
		for j, id := range batch {
			idStrs[j] = strconv.Itoa(id)
		}

		url := c.baseURL + "/subjects?types=vocabulary&ids=" + strings.Join(idStrs, ",")
		for url != "" {
			var resp subjectsResponse
			if err := c.doGet(ctx, apiKey, url, &resp); err != nil {
				return nil, err
			}
			for _, item := range resp.Data {
				if item.Data.Characters != "" {
					items = append(items, VocabItem{
						SubjectID:  item.ID,
						Characters: item.Data.Characters,
					})
				}
			}
			url = resp.Pages.NextURL
		}
	}

	return items, nil
}

func (c *wanikaniClient) doGet(ctx context.Context, apiKey, url string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating WaniKani request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Wanikani-Revision", "20170710")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling WaniKani API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("WaniKani API: invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WaniKani API: unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding WaniKani response: %w", err)
	}

	return nil
}

type paginationPages struct {
	NextURL string `json:"next_url"`
}

type assignmentsResponse struct {
	Pages paginationPages  `json:"pages"`
	Data  []assignmentItem `json:"data"`
}

type assignmentItem struct {
	Data assignmentData `json:"data"`
}

type assignmentData struct {
	SubjectID int `json:"subject_id"`
}

type subjectsResponse struct {
	Pages paginationPages `json:"pages"`
	Data  []subjectItem   `json:"data"`
}

type subjectItem struct {
	ID   int         `json:"id"`
	Data subjectData `json:"data"`
}

type subjectData struct {
	Characters string `json:"characters"`
}
