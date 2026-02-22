package domain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

var ErrVocabNotFound = errors.New("vocab entry not found")
var ErrUserNotFound = errors.New("user not found")

var DefaultUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type User struct {
	ID          uuid.UUID
	Handle      string
	GoogleSub   *string
	Email       *string
	Name        *string
	AvatarURL   *string
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type UserSettings struct {
	UserID               uuid.UUID
	JLPTLevel            string
	StoryWordTarget      int
	WaniKaniAPIKey       *string
	WaniKaniLastSyncedAt *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type VocabEntry struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Surface           string
	NormalizedSurface string
	Meaning           *string
	Reading           *string
	Source            string
	SourceRef         *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NormalizeSurface(s string) string {
	return strings.TrimSpace(norm.NFKC.String(s))
}

//go:generate go run go.uber.org/mock/mockgen -destination mock/mock_user_repository.go -package mock . UserRepository

type UserRepository interface {
	UpsertGoogleUser(ctx context.Context, googleSub, email, name, avatarURL string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	EnsureUserSettings(ctx context.Context, userID uuid.UUID) error
}

//go:generate go run go.uber.org/mock/mockgen -destination mock/mock_settings_repository.go -package mock . SettingsRepository

type SettingsRepository interface {
	Get(ctx context.Context, userID uuid.UUID) (*UserSettings, error)
	Update(ctx context.Context, userID uuid.UUID, jlptLevel string, storyWordTarget int, wanikaniAPIKey *string) (*UserSettings, error)
	UpdateWaniKaniSyncedAt(ctx context.Context, userID uuid.UUID, syncedAt time.Time) error
}

//go:generate go run go.uber.org/mock/mockgen -destination mock/mock_vocab_repository.go -package mock . VocabRepository

type VocabRepository interface {
	List(ctx context.Context, userID uuid.UUID, query string, limit, offset int) ([]VocabEntry, int, error)
	BatchUpsert(ctx context.Context, userID uuid.UUID, surfaces []string, source string) ([]VocabEntry, error)
	GetByID(ctx context.Context, id uuid.UUID) (*VocabEntry, error)
	UpdateDetails(ctx context.Context, id uuid.UUID, meaning, reading string) error
	GetByNormalizedSurfaces(ctx context.Context, userID uuid.UUID, surfaces []string) ([]VocabEntry, error)
}
