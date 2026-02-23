package config

import "time"

func (c *Config) ElevenLabsVoiceMap() map[string]string {
	return map[string]string{
		"N5_funny_male":      c.ElevenLabsVoiceIDN5FunnyM,
		"N5_funny_female":    c.ElevenLabsVoiceIDN5FunnyF,
		"N5_shocking_male":   c.ElevenLabsVoiceIDN5ShockingM,
		"N5_shocking_female": c.ElevenLabsVoiceIDN5ShockingF,
		"N4_funny_male":      c.ElevenLabsVoiceIDN4FunnyM,
		"N4_funny_female":    c.ElevenLabsVoiceIDN4FunnyF,
		"N4_shocking_male":   c.ElevenLabsVoiceIDN4ShockingM,
		"N4_shocking_female": c.ElevenLabsVoiceIDN4ShockingF,
		"N3_funny_male":      c.ElevenLabsVoiceIDN3FunnyM,
	}
}

type Config struct {
	GoogleClientID    string        `env:"GOOGLE_CLIENT_ID,required"`
	SessionSecret     string        `env:"SESSION_SECRET,required"`
	SessionCookieName string        `env:"SESSION_COOKIE_NAME,default=shiru_session"`
	SessionTTL        time.Duration `env:"SESSION_TTL,default=72h"`
	CookieSecure      bool          `env:"COOKIE_SECURE,default=false"`
	AllowedOrigin     string        `env:"ALLOWED_ORIGIN,required"`
	DatabaseURL          string `env:"DATABASE_URL,required"`
	ElasticsearchURL     string `env:"ELASTICSEARCH_URL,required"`
	AnthropicAPIKey      string `env:"ANTHROPIC_API_KEY,required"`
	AnthropicModel       string `env:"ANTHROPIC_MODEL,default=claude-sonnet-4-20250514"`
	ElevenLabsAPIKey              string `env:"ELEVENLABS_API_KEY"`
	ElevenLabsVoiceIDDefault      string `env:"ELEVENLABS_VOICE_ID_DEFAULT"`
	ElevenLabsVoiceIDN5FunnyM     string `env:"ELEVENLABS_VOICE_ID_N5_FUNNY_MALE"`
	ElevenLabsVoiceIDN5FunnyF     string `env:"ELEVENLABS_VOICE_ID_N5_FUNNY_FEMALE"`
	ElevenLabsVoiceIDN5ShockingM  string `env:"ELEVENLABS_VOICE_ID_N5_SHOCKING_MALE"`
	ElevenLabsVoiceIDN5ShockingF  string `env:"ELEVENLABS_VOICE_ID_N5_SHOCKING_FEMALE"`
	ElevenLabsVoiceIDN4FunnyM     string `env:"ELEVENLABS_VOICE_ID_N4_FUNNY_MALE"`
	ElevenLabsVoiceIDN4FunnyF     string `env:"ELEVENLABS_VOICE_ID_N4_FUNNY_FEMALE"`
	ElevenLabsVoiceIDN4ShockingM  string `env:"ELEVENLABS_VOICE_ID_N4_SHOCKING_MALE"`
	ElevenLabsVoiceIDN4ShockingF  string `env:"ELEVENLABS_VOICE_ID_N4_SHOCKING_FEMALE"`
	ElevenLabsVoiceIDN3FunnyM     string `env:"ELEVENLABS_VOICE_ID_N3_FUNNY_MALE"`
	DictionaryAPIBaseURL string `env:"DICTIONARY_API_BASE_URL"`
	WaniKaniAPIBaseURL   string `env:"WANIKANI_API_BASE_URL,default=https://api.wanikani.com/v2"`
	S3Endpoint  string `env:"S3_ENDPOINT,required"`
	S3Bucket    string `env:"S3_BUCKET,required"`
	S3AccessKey string `env:"S3_ACCESS_KEY,required"`
	S3SecretKey string `env:"S3_SECRET_KEY,required"`
	S3UseSSL    bool   `env:"S3_USE_SSL,default=false"`
}
