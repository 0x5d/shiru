package story

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate go run go.uber.org/mock/mockgen -destination mock/repository.go -package mock . Repository

type Repository interface {
	Create(ctx context.Context, story *Story) error
	Get(ctx context.Context, userID, id uuid.UUID) (*Story, error)
	List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Story, error)
	AddVocabEntries(ctx context.Context, storyID uuid.UUID, vocabEntryIDs []uuid.UUID) error
}

var _ Repository = (*PostgresRepository)(nil)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, story *Story) error {
	now := time.Now()
	if story.ID == uuid.Nil {
		story.ID = uuid.New()
	}
	story.CreatedAt = now
	story.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO stories (id, user_id, topic, title, tone, jlpt_level, target_word_count, actual_word_count, content, used_vocab_count, source_tag_names, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		story.ID, story.UserID, story.Topic, story.Title, story.Tone,
		story.JLPTLevel, story.TargetWordCount, story.ActualWordCount,
		story.Content, story.UsedVocabCount, story.SourceTagNames,
		story.CreatedAt, story.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating story: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Get(ctx context.Context, userID, id uuid.UUID) (*Story, error) {
	var s Story
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, topic, title, tone, jlpt_level, target_word_count, actual_word_count, content, used_vocab_count, source_tag_names, created_at, updated_at
		FROM stories WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(
		&s.ID, &s.UserID, &s.Topic, &s.Title, &s.Tone,
		&s.JLPTLevel, &s.TargetWordCount, &s.ActualWordCount,
		&s.Content, &s.UsedVocabCount, &s.SourceTagNames,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting story: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Story, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, topic, title, tone, jlpt_level, target_word_count, actual_word_count, content, used_vocab_count, source_tag_names, created_at, updated_at
		FROM stories WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("listing stories: %w", err)
	}
	defer rows.Close()

	var stories []*Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Topic, &s.Title, &s.Tone,
			&s.JLPTLevel, &s.TargetWordCount, &s.ActualWordCount,
			&s.Content, &s.UsedVocabCount, &s.SourceTagNames,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning story: %w", err)
		}
		stories = append(stories, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating stories: %w", err)
	}
	return stories, nil
}

func (r *PostgresRepository) AddVocabEntries(ctx context.Context, storyID uuid.UUID, vocabEntryIDs []uuid.UUID) error {
	if len(vocabEntryIDs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, id := range vocabEntryIDs {
		batch.Queue(
			`INSERT INTO story_vocab_entries (story_id, vocab_entry_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			storyID, id,
		)
	}
	results := r.pool.SendBatch(ctx, batch)
	defer func() { _ = results.Close() }()

	for range vocabEntryIDs {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("adding vocab entry to story: %w", err)
		}
	}
	return nil
}
