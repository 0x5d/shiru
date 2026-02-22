package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveOverlapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  []Token
		expect []Token
	}{
		{
			name:   "empty",
			input:  []Token{},
			expect: []Token{},
		},
		{
			name: "no overlap",
			input: []Token{
				{Surface: "たろう", StartOffset: 0, EndOffset: 3},
				{Surface: "くん", StartOffset: 3, EndOffset: 5},
			},
			expect: []Token{
				{Surface: "たろう", StartOffset: 0, EndOffset: 3},
				{Surface: "くん", StartOffset: 3, EndOffset: 5},
			},
		},
		{
			name: "compound decomposition keeps longest",
			input: []Token{
				{Surface: "放課後", StartOffset: 0, EndOffset: 3},
				{Surface: "放課", StartOffset: 0, EndOffset: 2},
				{Surface: "後", StartOffset: 2, EndOffset: 3},
			},
			expect: []Token{
				{Surface: "放課後", StartOffset: 0, EndOffset: 3},
			},
		},
		{
			name: "sub-tokens listed first still picks compound",
			input: []Token{
				{Surface: "放課", StartOffset: 0, EndOffset: 2},
				{Surface: "放課後", StartOffset: 0, EndOffset: 3},
				{Surface: "後", StartOffset: 2, EndOffset: 3},
			},
			expect: []Token{
				{Surface: "放課後", StartOffset: 0, EndOffset: 3},
			},
		},
		{
			name: "compound in middle of sentence",
			input: []Token{
				{Surface: "今日", StartOffset: 0, EndOffset: 2},
				{Surface: "の", StartOffset: 2, EndOffset: 3},
				{Surface: "放課後", StartOffset: 3, EndOffset: 6},
				{Surface: "放課", StartOffset: 3, EndOffset: 5},
				{Surface: "後", StartOffset: 5, EndOffset: 6},
			},
			expect: []Token{
				{Surface: "今日", StartOffset: 0, EndOffset: 2},
				{Surface: "の", StartOffset: 2, EndOffset: 3},
				{Surface: "放課後", StartOffset: 3, EndOffset: 6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := removeOverlapping(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}
