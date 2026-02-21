package story_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	shiruanthropic "github.com/0x5d/shiru/internal/anthropic"
	anthropicmock "github.com/0x5d/shiru/internal/anthropic/mock"
	"github.com/0x5d/shiru/internal/story"
	storymock "github.com/0x5d/shiru/internal/story/mock"
)

func TestService_Generate(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	vocabID1 := uuid.New()
	vocabID2 := uuid.New()

	tests := []struct {
		name    string
		params  story.GenerateParams
		setup   func(ac *anthropicmock.MockClient, sr *storymock.MockRepository, vr *storymock.MockVocabRepository)
		wantErr string
	}{
		{
			name: "successful generation",
			params: story.GenerateParams{
				UserID:          userID,
				Topic:           "夏祭り",
				Tags:            []string{"nature", "food", "family"},
				JLPTLevel:       "N4",
				TargetWordCount: 100,
			},
			setup: func(ac *anthropicmock.MockClient, sr *storymock.MockRepository, vr *storymock.MockVocabRepository) {
				ac.EXPECT().
					RankTags(gomock.Any(), "夏祭り", []string{"nature", "food", "family"}).
					Return([]string{"nature", "food"}, nil)

				vr.EXPECT().
					ListByTags(gomock.Any(), userID, []string{"nature", "food"}, 50).
					Return([]story.VocabEntry{
						{ID: vocabID1, Surface: "花"},
						{ID: vocabID2, Surface: "食べる"},
					}, nil)

				ac.EXPECT().
					GenerateStory(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, p shiruanthropic.StoryParams) (*shiruanthropic.StoryResult, error) {
						assert.Equal(t, "夏祭り", p.Topic)
						assert.Equal(t, "N4", p.JLPTLevel)
						assert.Equal(t, 100, p.TargetWordCount)
						assert.ElementsMatch(t, []string{"花", "食べる"}, p.Vocab)
						assert.Contains(t, []string{"funny", "shocking"}, p.Tone)
						return &shiruanthropic.StoryResult{
							Title: "夏祭りの日",
							Story: "花がきれいでした。食べるものがたくさんありました。",
						}, nil
					})

				sr.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, s *story.Story) error {
						assert.Equal(t, userID, s.UserID)
						assert.Equal(t, "夏祭り", s.Topic)
						assert.Equal(t, "夏祭りの日", s.Title)
						assert.Equal(t, "N4", s.JLPTLevel)
						assert.Equal(t, 100, s.TargetWordCount)
						assert.Equal(t, 2, s.UsedVocabCount)
						assert.Equal(t, []string{"nature", "food"}, s.SourceTagNames)
						s.ID = uuid.New()
						return nil
					})

				sr.EXPECT().
					AddVocabEntries(gomock.Any(), gomock.Any(), gomock.Eq([]uuid.UUID{vocabID1, vocabID2})).
					Return(nil)
			},
		},
		{
			name: "no tags available",
			params: story.GenerateParams{
				UserID:          userID,
				Topic:           "夏祭り",
				Tags:            []string{},
				JLPTLevel:       "N4",
				TargetWordCount: 100,
			},
			setup:   func(_ *anthropicmock.MockClient, _ *storymock.MockRepository, _ *storymock.MockVocabRepository) {},
			wantErr: "no tags available",
		},
		{
			name: "rank tags error",
			params: story.GenerateParams{
				UserID:          userID,
				Topic:           "夏祭り",
				Tags:            []string{"nature"},
				JLPTLevel:       "N4",
				TargetWordCount: 100,
			},
			setup: func(ac *anthropicmock.MockClient, _ *storymock.MockRepository, _ *storymock.MockVocabRepository) {
				ac.EXPECT().
					RankTags(gomock.Any(), "夏祭り", []string{"nature"}).
					Return(nil, fmt.Errorf("API error"))
			},
			wantErr: "ranking tags: API error",
		},
		{
			name: "vocab listing error",
			params: story.GenerateParams{
				UserID:          userID,
				Topic:           "夏祭り",
				Tags:            []string{"nature"},
				JLPTLevel:       "N4",
				TargetWordCount: 100,
			},
			setup: func(ac *anthropicmock.MockClient, _ *storymock.MockRepository, vr *storymock.MockVocabRepository) {
				ac.EXPECT().
					RankTags(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]string{"nature"}, nil)

				vr.EXPECT().
					ListByTags(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("db error"))
			},
			wantErr: "listing vocab by tags: db error",
		},
		{
			name: "story generation error",
			params: story.GenerateParams{
				UserID:          userID,
				Topic:           "夏祭り",
				Tags:            []string{"nature"},
				JLPTLevel:       "N4",
				TargetWordCount: 100,
			},
			setup: func(ac *anthropicmock.MockClient, _ *storymock.MockRepository, vr *storymock.MockVocabRepository) {
				ac.EXPECT().
					RankTags(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]string{"nature"}, nil)

				vr.EXPECT().
					ListByTags(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]story.VocabEntry{{ID: vocabID1, Surface: "花"}}, nil)

				ac.EXPECT().
					GenerateStory(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("LLM error"))
			},
			wantErr: "generating story: LLM error",
		},
		{
			name: "story persistence error",
			params: story.GenerateParams{
				UserID:          userID,
				Topic:           "夏祭り",
				Tags:            []string{"nature"},
				JLPTLevel:       "N4",
				TargetWordCount: 100,
			},
			setup: func(ac *anthropicmock.MockClient, sr *storymock.MockRepository, vr *storymock.MockVocabRepository) {
				ac.EXPECT().
					RankTags(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]string{"nature"}, nil)

				vr.EXPECT().
					ListByTags(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]story.VocabEntry{{ID: vocabID1, Surface: "花"}}, nil)

				ac.EXPECT().
					GenerateStory(gomock.Any(), gomock.Any()).
					Return(&shiruanthropic.StoryResult{Title: "タイトル", Story: "話"}, nil)

				sr.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("db write error"))
			},
			wantErr: "persisting story: db write error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			ac := anthropicmock.NewMockClient(ctrl)
			sr := storymock.NewMockRepository(ctrl)
			vr := storymock.NewMockVocabRepository(ctrl)

			tt.setup(ac, sr, vr)

			svc := story.NewService(ac, sr, vr, nil, logr.Discard())
			s, err := svc.Generate(context.Background(), tt.params)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, s)
			assert.Equal(t, userID, s.UserID)
			assert.Equal(t, tt.params.Topic, s.Topic)
		})
	}
}

func TestService_GenerateTopics(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	ac := anthropicmock.NewMockClient(ctrl)
	sr := storymock.NewMockRepository(ctrl)
	vr := storymock.NewMockVocabRepository(ctrl)

	expected := []string{"夏の庭", "料理の冒険", "花見"}
	ac.EXPECT().
		GenerateTopics(gomock.Any(), []string{"nature", "food"}, "N4").
		Return(expected, nil)

	svc := story.NewService(ac, sr, vr, nil, logr.Discard())
	topics, err := svc.GenerateTopics(context.Background(), []string{"nature", "food"}, "N4")
	require.NoError(t, err)
	assert.Equal(t, expected, topics)
}
