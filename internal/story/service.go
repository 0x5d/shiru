package story

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	shiruanthropic "github.com/0x5d/shiru/internal/anthropic"
)

//go:generate go run go.uber.org/mock/mockgen -destination mock/vocab_repository.go -package mock . VocabRepository

type VocabEntry struct {
	ID      uuid.UUID
	Surface string
}

type VocabRepository interface {
	ListByTags(ctx context.Context, userID uuid.UUID, tagNames []string, limit int) ([]VocabEntry, error)
}

//go:generate go run go.uber.org/mock/mockgen -destination mock/indexer.go -package mock . Indexer

type Indexer interface {
	Index(ctx context.Context, story *Story) error
}

type GenerateParams struct {
	UserID          uuid.UUID
	Topic           string
	Tags            []string
	JLPTLevel       string
	TargetWordCount int
}

type Service struct {
	anthropic shiruanthropic.Client
	stories   Repository
	vocab     VocabRepository
	indexer   Indexer
	log       logr.Logger
}

func NewService(anthropic shiruanthropic.Client, stories Repository, vocab VocabRepository, indexer Indexer, log logr.Logger) *Service {
	return &Service{
		anthropic: anthropic,
		stories:   stories,
		vocab:     vocab,
		indexer:   indexer,
		log:       log,
	}
}

func (s *Service) GenerateTopics(ctx context.Context, tags []string, jlptLevel string) ([]string, error) {
	topics, err := s.anthropic.GenerateTopics(ctx, tags, jlptLevel)
	if err != nil {
		return nil, fmt.Errorf("generating topics: %w", err)
	}
	return topics, nil
}

func (s *Service) Generate(ctx context.Context, params GenerateParams) (*Story, error) {
	if len(params.Tags) == 0 {
		return nil, fmt.Errorf("no tags available for story generation")
	}

	topTags, err := s.anthropic.RankTags(ctx, params.Topic, params.Tags)
	if err != nil {
		return nil, fmt.Errorf("ranking tags: %w", err)
	}
	s.log.Info("ranked tags for story", "topic", params.Topic, "top_tags", topTags)

	vocabEntries, err := s.vocab.ListByTags(ctx, params.UserID, topTags, 50)
	if err != nil {
		return nil, fmt.Errorf("listing vocab by tags: %w", err)
	}
	s.log.Info("selected vocab for story", "count", len(vocabEntries))

	surfaces := make([]string, len(vocabEntries))
	for i, v := range vocabEntries {
		surfaces[i] = v.Surface
	}

	tone := randomTone()

	result, err := s.anthropic.GenerateStory(ctx, shiruanthropic.StoryParams{
		Topic:           params.Topic,
		Vocab:           surfaces,
		JLPTLevel:       params.JLPTLevel,
		Tone:            tone,
		TargetWordCount: params.TargetWordCount,
	})
	if err != nil {
		return nil, fmt.Errorf("generating story: %w", err)
	}

	wordCount := countWords(result.Story)

	story := &Story{
		UserID:          params.UserID,
		Topic:           params.Topic,
		Title:           result.Title,
		Tone:            tone,
		JLPTLevel:       params.JLPTLevel,
		TargetWordCount: params.TargetWordCount,
		ActualWordCount: wordCount,
		Content:         result.Story,
		UsedVocabCount:  len(vocabEntries),
		SourceTagNames:  topTags,
	}

	if err := s.stories.Create(ctx, story); err != nil {
		return nil, fmt.Errorf("persisting story: %w", err)
	}

	vocabIDs := make([]uuid.UUID, len(vocabEntries))
	for i, v := range vocabEntries {
		vocabIDs[i] = v.ID
	}
	if err := s.stories.AddVocabEntries(ctx, story.ID, vocabIDs); err != nil {
		return nil, fmt.Errorf("adding vocab entries to story: %w", err)
	}

	if s.indexer != nil {
		if err := s.indexer.Index(ctx, story); err != nil {
			s.log.Error(err, "indexing story", "story_id", story.ID)
		}
	}

	return story, nil
}

func randomTone() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(2))
	if n.Int64() == 0 {
		return "funny"
	}
	return "shocking"
}

func countWords(text string) int {
	return len(strings.Fields(text))
}
