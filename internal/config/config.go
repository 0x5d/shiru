package config

import "time"

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
	ElevenLabsAPIKey     string `env:"ELEVENLABS_API_KEY"`
	ElevenLabsVoiceID    string `env:"ELEVENLABS_VOICE_ID"`
	DictionaryAPIBaseURL string `env:"DICTIONARY_API_BASE_URL"`
	WaniKaniAPIBaseURL   string `env:"WANIKANI_API_BASE_URL,default=https://api.wanikani.com/v2"`
	S3Endpoint  string `env:"S3_ENDPOINT,required"`
	S3Bucket    string `env:"S3_BUCKET,required"`
	S3AccessKey string `env:"S3_ACCESS_KEY,required"`
	S3SecretKey string `env:"S3_SECRET_KEY,required"`
	S3UseSSL    bool   `env:"S3_USE_SSL,default=false"`
}
