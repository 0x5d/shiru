package audio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("story audio not found")

type StoryAudio struct {
	StoryID     uuid.UUID
	VoiceID     string
	AudioFormat string
	StoragePath string
	DurationMS  *int
	Checksum    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

//go:generate go run go.uber.org/mock/mockgen -destination mock/repository.go -package mock . Repository

type Repository interface {
	GetByStoryID(ctx context.Context, storyID uuid.UUID) (*StoryAudio, error)
	Create(ctx context.Context, audio *StoryAudio) error
}

var _ Repository = (*PostgresRepository)(nil)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetByStoryID(ctx context.Context, storyID uuid.UUID) (*StoryAudio, error) {
	var a StoryAudio
	err := r.pool.QueryRow(ctx, `
		SELECT story_id, voice_id, audio_format, storage_path, duration_ms, checksum, created_at, updated_at
		FROM story_audio WHERE story_id = $1`, storyID,
	).Scan(&a.StoryID, &a.VoiceID, &a.AudioFormat, &a.StoragePath, &a.DurationMS, &a.Checksum, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting story audio: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) Create(ctx context.Context, audio *StoryAudio) error {
	now := time.Now()
	audio.CreatedAt = now
	audio.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO story_audio (story_id, voice_id, audio_format, storage_path, duration_ms, checksum, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		audio.StoryID, audio.VoiceID, audio.AudioFormat, audio.StoragePath, audio.DurationMS, audio.Checksum, audio.CreatedAt, audio.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating story audio: %w", err)
	}
	return nil
}

//go:generate go run go.uber.org/mock/mockgen -destination mock/file_store.go -package mock . FileStore

type FileStore interface {
	Write(path string, data []byte) error
	Read(path string) ([]byte, error)
}

var _ FileStore = (*DiskFileStore)(nil)

type DiskFileStore struct {
	basePath string
}

func NewDiskFileStore(basePath string) *DiskFileStore {
	return &DiskFileStore{basePath: basePath}
}

func (s *DiskFileStore) Write(path string, data []byte) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return fmt.Errorf("creating audio directory: %w", err)
	}
	if err := os.WriteFile(fullPath, data, 0o640); err != nil {
		return fmt.Errorf("writing audio file: %w", err)
	}
	return nil
}

func (s *DiskFileStore) Read(path string) ([]byte, error) {
	fullPath := filepath.Join(s.basePath, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading audio file: %w", err)
	}
	return data, nil
}
