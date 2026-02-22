package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TopicSnapshot struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Topics    []string
	CreatedAt time.Time
}

var ErrNoSnapshot = errors.New("no topic snapshot found")

type TopicSnapshotRepository struct {
	pool *pgxpool.Pool
}

func NewTopicSnapshotRepository(pool *pgxpool.Pool) *TopicSnapshotRepository {
	return &TopicSnapshotRepository{pool: pool}
}

func (r *TopicSnapshotRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*TopicSnapshot, error) {
	var s TopicSnapshot
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, topics, created_at
		FROM topic_snapshots
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&s.ID, &s.UserID, &s.Topics, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoSnapshot
		}
		return nil, fmt.Errorf("getting latest topic snapshot: %w", err)
	}
	return &s, nil
}

func (r *TopicSnapshotRepository) Create(ctx context.Context, userID uuid.UUID, topics []string) (*TopicSnapshot, error) {
	s := TopicSnapshot{
		ID:        uuid.New(),
		UserID:    userID,
		Topics:    topics,
		CreatedAt: time.Now(),
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO topic_snapshots (id, user_id, topics, prompt_version, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, s.ID, s.UserID, s.Topics, "v1", s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating topic snapshot: %w", err)
	}
	return &s, nil
}
