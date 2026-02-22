package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/0x5d/shiru/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VocabRepository struct {
	pool *pgxpool.Pool
}

var _ domain.VocabRepository = &VocabRepository{}

func NewVocabRepository(pool *pgxpool.Pool) *VocabRepository {
	return &VocabRepository{pool: pool}
}

func (r *VocabRepository) List(ctx context.Context, userID uuid.UUID, query string, limit, offset int) ([]domain.VocabEntry, int, error) {
	var total int
	countArgs := []any{userID}
	countQuery := `SELECT COUNT(*) FROM vocab_entries WHERE user_id = $1`
	if query != "" {
		countQuery += ` AND normalized_surface LIKE $2`
		countArgs = append(countArgs, "%"+domain.NormalizeSurface(query)+"%")
	}
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting vocab: %w", err)
	}

	listQuery := `SELECT id, user_id, surface, normalized_surface, meaning, reading, source, source_ref, created_at, updated_at
		FROM vocab_entries WHERE user_id = $1`
	listArgs := []any{userID}
	argIdx := 2
	if query != "" {
		listQuery += fmt.Sprintf(` AND normalized_surface LIKE $%d`, argIdx)
		listArgs = append(listArgs, "%"+domain.NormalizeSurface(query)+"%")
		argIdx++
	}
	listQuery += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing vocab: %w", err)
	}
	defer rows.Close()

	var entries []domain.VocabEntry
	for rows.Next() {
		var e domain.VocabEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Surface, &e.NormalizedSurface, &e.Meaning, &e.Reading, &e.Source, &e.SourceRef, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning vocab entry: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, total, nil
}

func (r *VocabRepository) BatchUpsert(ctx context.Context, userID uuid.UUID, surfaces []string, source string) ([]domain.VocabEntry, error) {
	seen := make(map[string]string)
	type pair struct{ surface, normalized string }
	var deduped []pair
	for _, s := range surfaces {
		n := domain.NormalizeSurface(s)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; !ok {
			seen[n] = s
			deduped = append(deduped, pair{s, n})
		}
	}

	if len(deduped) == 0 {
		return nil, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var entries []domain.VocabEntry
	for _, d := range deduped {
		var e domain.VocabEntry
		err := tx.QueryRow(ctx, `
			INSERT INTO vocab_entries (id, user_id, surface, normalized_surface, source)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id, normalized_surface) DO UPDATE SET updated_at = NOW()
			RETURNING id, user_id, surface, normalized_surface, meaning, reading, source, source_ref, created_at, updated_at
		`, uuid.New(), userID, d.surface, d.normalized, source).Scan(
			&e.ID, &e.UserID, &e.Surface, &e.NormalizedSurface, &e.Meaning, &e.Reading, &e.Source, &e.SourceRef, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("upserting vocab entry %q: %w", d.surface, err)
		}
		entries = append(entries, e)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return entries, nil
}

func (r *VocabRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.VocabEntry, error) {
	var e domain.VocabEntry
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, surface, normalized_surface, meaning, reading, source, source_ref, created_at, updated_at
		FROM vocab_entries WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&e.ID, &e.UserID, &e.Surface, &e.NormalizedSurface, &e.Meaning, &e.Reading, &e.Source, &e.SourceRef, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVocabNotFound
		}
		return nil, fmt.Errorf("getting vocab entry: %w", err)
	}
	return &e, nil
}

func (r *VocabRepository) UpdateDetails(ctx context.Context, userID, id uuid.UUID, meaning, reading string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE vocab_entries SET meaning = $3, reading = $4, updated_at = NOW()
		WHERE id = $1 AND user_id = $2`, id, userID, meaning, reading,
	)
	if err != nil {
		return fmt.Errorf("updating vocab details: %w", err)
	}
	return nil
}

func (r *VocabRepository) GetByNormalizedSurfaces(ctx context.Context, userID uuid.UUID, surfaces []string) ([]domain.VocabEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, surface, normalized_surface, meaning, reading, source, source_ref, created_at, updated_at
		FROM vocab_entries
		WHERE user_id = $1 AND normalized_surface = ANY($2)`, userID, surfaces,
	)
	if err != nil {
		return nil, fmt.Errorf("getting vocab by surfaces: %w", err)
	}
	defer rows.Close()

	var entries []domain.VocabEntry
	for rows.Next() {
		var e domain.VocabEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Surface, &e.NormalizedSurface, &e.Meaning, &e.Reading, &e.Source, &e.SourceRef, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning vocab entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}
