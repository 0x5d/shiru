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

type UserRepository struct {
	pool *pgxpool.Pool
}

var _ domain.UserRepository = &UserRepository{}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) UpsertGoogleUser(ctx context.Context, googleSub, email, name, avatarURL string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (id, handle, google_sub, email, name, avatar_url, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (google_sub) WHERE google_sub IS NOT NULL
		DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			last_login_at = NOW(),
			updated_at = NOW()
		RETURNING id, handle, google_sub, email, name, avatar_url, last_login_at, created_at, updated_at
	`, uuid.New(), email, googleSub, email, name, avatarURL).Scan(
		&u.ID, &u.Handle, &u.GoogleSub, &u.Email, &u.Name, &u.AvatarURL, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upserting google user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, handle, google_sub, email, name, avatar_url, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Handle, &u.GoogleSub, &u.Email, &u.Name, &u.AvatarURL, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("getting user by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) EnsureUserSettings(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_settings (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	if err != nil {
		return fmt.Errorf("ensuring user settings: %w", err)
	}
	return nil
}
