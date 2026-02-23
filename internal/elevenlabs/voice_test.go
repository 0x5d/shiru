package elevenlabs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVoiceSelector_Select(t *testing.T) {
	t.Parallel()

	voices := map[string]string{
		"N5_funny_male":      "voice-n5-funny-m",
		"N5_funny_female":    "voice-n5-funny-f",
		"N5_shocking_male":   "voice-n5-shocking-m",
		"N4_funny_male":      "voice-n4-funny-m",
		"N4_shocking_female": "voice-n4-shocking-f",
	}

	tests := []struct {
		name   string
		level  string
		tone   string
		gender string
		want   string
	}{
		{
			name:   "exact match male",
			level:  "N5",
			tone:   "funny",
			gender: "male",
			want:   "voice-n5-funny-m",
		},
		{
			name:   "exact match female",
			level:  "N5",
			tone:   "funny",
			gender: "female",
			want:   "voice-n5-funny-f",
		},
		{
			name:   "falls back to other gender",
			level:  "N4",
			tone:   "shocking",
			gender: "male",
			want:   "voice-n4-shocking-f",
		},
		{
			name:   "falls back to default when no match",
			level:  "N1",
			tone:   "funny",
			gender: "male",
			want:   "default-voice",
		},
		{
			name:   "falls back to default for missing tone",
			level:  "N5",
			tone:   "mysterious",
			gender: "male",
			want:   "default-voice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vs := NewVoiceSelector("default-voice", voices)
			vs.randGender = func() string { return tt.gender }
			got := vs.Select(tt.level, tt.tone)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVoiceSelector_SelectNormalizesCase(t *testing.T) {
	t.Parallel()

	voices := map[string]string{
		"N5_funny_male": "voice-n5-funny-m",
	}
	vs := NewVoiceSelector("default-voice", voices)
	vs.randGender = func() string { return "male" }

	assert.Equal(t, "voice-n5-funny-m", vs.Select("n5", "Funny"))
	assert.Equal(t, "voice-n5-funny-m", vs.Select("N5", "FUNNY"))
	assert.Equal(t, "voice-n5-funny-m", vs.Select("N5", "funny"))
}

func TestNewVoiceSelector_SkipsEmptyAndMalformed(t *testing.T) {
	t.Parallel()

	voices := map[string]string{
		"N5_funny_male": "voice-id",
		"bad_key":       "ignored",
		"N4_funny_male": "",
	}
	vs := NewVoiceSelector("default", voices)
	assert.Len(t, vs.voices, 1)
}

func TestNewVoiceSelector_ConfigVoiceMapKeysAllParse(t *testing.T) {
	t.Parallel()

	configKeys := []string{
		"N5_funny_male",
		"N5_funny_female",
		"N5_shocking_male",
		"N5_shocking_female",
		"N4_funny_male",
		"N4_funny_female",
		"N4_shocking_male",
		"N4_shocking_female",
		"N3_funny_male",
	}
	voices := make(map[string]string, len(configKeys))
	for _, k := range configKeys {
		voices[k] = "voice-" + k
	}
	vs := NewVoiceSelector("default", voices)
	assert.Len(t, vs.voices, len(configKeys))
}
