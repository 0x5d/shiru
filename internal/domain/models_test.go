package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeSurface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain kanji", input: "花", want: "花"},
		{name: "trims whitespace", input: "  走る  ", want: "走る"},
		{name: "nfkc fullwidth to ascii", input: "コンピューター", want: "コンピューター"},
		{name: "nfkc halfwidth katakana", input: "ｺﾝﾋﾟｭｰﾀｰ", want: "コンピューター"},
		{name: "empty string", input: "", want: ""},
		{name: "whitespace only", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, NormalizeSurface(tt.input))
		})
	}
}
