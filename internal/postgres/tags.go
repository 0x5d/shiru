package postgres

import (
	"context"
	"fmt"

	"github.com/0x5d/shiru/internal/story"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ story.VocabRepository = (*TagRepository)(nil)

type TagRepository struct {
	pool *pgxpool.Pool
}

func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{pool: pool}
}

func (r *TagRepository) ListUserTags(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT name FROM tags WHERE user_id = $1 ORDER BY name`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning tag: %w", err)
		}
		tags = append(tags, name)
	}
	return tags, nil
}

func (r *TagRepository) ListByTags(ctx context.Context, userID uuid.UUID, tagNames []string, limit int) ([]story.VocabEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ve.id, ve.surface
		FROM vocab_entries ve
		JOIN vocab_entry_tags vet ON vet.vocab_entry_id = ve.id
		JOIN tags t ON t.id = vet.tag_id
		WHERE ve.user_id = $1 AND t.name = ANY($2)
		LIMIT $3`, userID, tagNames, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing vocab by tags: %w", err)
	}
	defer rows.Close()

	var entries []story.VocabEntry
	for rows.Next() {
		var e story.VocabEntry
		if err := rows.Scan(&e.ID, &e.Surface); err != nil {
			return nil, fmt.Errorf("scanning vocab entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}
