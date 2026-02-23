package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/0x5d/shiru/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepository struct {
	pool *pgxpool.Pool
}

var _ domain.SettingsRepository = &SettingsRepository{}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

func (r *SettingsRepository) Get(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	var s domain.UserSettings
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, jlpt_level, story_word_target, wanikani_api_key, wanikani_last_synced_at, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
	`, userID).Scan(
		&s.UserID, &s.JLPTLevel, &s.StoryWordTarget, &s.WaniKaniAPIKey, &s.WaniKaniLastSyncedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("querying settings: %w", err)
	}
	return &s, nil
}

func (r *SettingsRepository) UpdateWaniKaniSyncedAt(ctx context.Context, userID uuid.UUID, syncedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE user_settings SET wanikani_last_synced_at = $2, updated_at = NOW()
		WHERE user_id = $1`, userID, syncedAt,
	)
	if err != nil {
		return fmt.Errorf("updating wanikani synced at: %w", err)
	}
	return nil
}

func (r *SettingsRepository) ResetWaniKaniSyncedAt(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE user_settings SET wanikani_last_synced_at = NULL, updated_at = NOW()
		WHERE user_id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("resetting wanikani synced at: %w", err)
	}
	return nil
}

func (r *SettingsRepository) Update(ctx context.Context, userID uuid.UUID, jlptLevel string, storyWordTarget int, wanikaniAPIKey *string) (*domain.UserSettings, error) {
	var s domain.UserSettings
	err := r.pool.QueryRow(ctx, `
		UPDATE user_settings
		SET jlpt_level = $2, story_word_target = $3, wanikani_api_key = $4, updated_at = NOW()
		WHERE user_id = $1
		RETURNING user_id, jlpt_level, story_word_target, wanikani_api_key, wanikani_last_synced_at, created_at, updated_at
	`, userID, jlptLevel, storyWordTarget, wanikaniAPIKey).Scan(
		&s.UserID, &s.JLPTLevel, &s.StoryWordTarget, &s.WaniKaniAPIKey, &s.WaniKaniLastSyncedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("updating settings: %w", err)
	}
	return &s, nil
}
