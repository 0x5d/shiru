package story

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("story not found")

type Story struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Topic           string
	Title           string
	Tone            string
	JLPTLevel       string
	TargetWordCount int
	ActualWordCount int
	Content         string
	UsedVocabCount  int
	SourceTagNames  []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
