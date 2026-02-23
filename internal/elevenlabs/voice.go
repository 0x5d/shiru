package elevenlabs

import (
	"crypto/rand"
	"log"
	"math/big"
	"strings"
)

type voiceKey struct {
	level  string
	tone   string
	gender string
}

type VoiceSelector struct {
	voices     map[voiceKey]string
	defaultID  string
	randGender func() string
}

func NewVoiceSelector(defaultID string, voices map[string]string) *VoiceSelector {
	if defaultID == "" {
		log.Println("WARNING: ELEVENLABS_VOICE_ID_DEFAULT is empty, voice selection will produce empty voiceIDs when no specific match is found")
	}
	m := make(map[voiceKey]string, len(voices))
	for composite, id := range voices {
		if id == "" {
			continue
		}
		parts := strings.SplitN(composite, "_", 3)
		if len(parts) != 3 {
			continue
		}
		m[voiceKey{
			level:  strings.ToUpper(parts[0]),
			tone:   strings.ToLower(parts[1]),
			gender: strings.ToLower(parts[2]),
		}] = id
	}
	return &VoiceSelector{
		voices:     m,
		defaultID:  defaultID,
		randGender: cryptoRandGender,
	}
}

func (vs *VoiceSelector) Select(level, tone string) string {
	level = strings.ToUpper(level)
	tone = strings.ToLower(tone)
	gender := vs.randGender()
	if id, ok := vs.voices[voiceKey{level: level, tone: tone, gender: gender}]; ok {
		return id
	}
	other := "female"
	if gender == "female" {
		other = "male"
	}
	if id, ok := vs.voices[voiceKey{level: level, tone: tone, gender: other}]; ok {
		return id
	}
	return vs.defaultID
}

func cryptoRandGender() string {
	n, err := rand.Int(rand.Reader, big.NewInt(2))
	if err != nil {
		return "male"
	}
	if n.Int64() == 0 {
		return "male"
	}
	return "female"
}
