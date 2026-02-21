package story

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomTone(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for range 100 {
		seen[randomTone()] = true
	}
	assert.True(t, seen["funny"], "expected funny tone to appear")
	assert.True(t, seen["shocking"], "expected shocking tone to appear")
}

func TestCountWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"花がきれいでした。 食べるものがたくさんありました。", 2},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, countWords(tt.text))
	}
}
