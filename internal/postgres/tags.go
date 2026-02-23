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

func (r *TagRepository) UpsertTagsAndLink(ctx context.Context, userID uuid.UUID, vocabEntryID uuid.UUID, tagNames []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for rank, name := range tagNames {
		var tagID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO tags (id, user_id, name)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, name) DO UPDATE SET updated_at = NOW()
			RETURNING id
		`, uuid.New(), userID, name).Scan(&tagID)
		if err != nil {
			return fmt.Errorf("upserting tag %q: %w", name, err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO vocab_entry_tags (vocab_entry_id, tag_id, rank)
			VALUES ($1, $2, $3)
			ON CONFLICT (vocab_entry_id, tag_id) DO NOTHING
		`, vocabEntryID, tagID, rank+1)
		if err != nil {
			return fmt.Errorf("linking vocab entry to tag: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *TagRepository) CountTaggedVocab(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT ve.id)
		FROM vocab_entries ve
		JOIN vocab_entry_tags vet ON vet.vocab_entry_id = ve.id
		WHERE ve.user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting tagged vocab: %w", err)
	}
	return count, nil
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
