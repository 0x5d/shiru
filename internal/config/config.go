package config

type Config struct {
	DatabaseURL          string `env:"DATABASE_URL,required"`
	ElasticsearchURL     string `env:"ELASTICSEARCH_URL,required"`
	AnthropicAPIKey      string `env:"ANTHROPIC_API_KEY,required"`
	AnthropicModel       string `env:"ANTHROPIC_MODEL,default=claude-sonnet-4-20250514"`
	ElevenLabsAPIKey     string `env:"ELEVENLABS_API_KEY"`
	ElevenLabsVoiceID    string `env:"ELEVENLABS_VOICE_ID"`
	DictionaryAPIBaseURL string `env:"DICTIONARY_API_BASE_URL"`
	WaniKaniAPIBaseURL   string `env:"WANIKANI_API_BASE_URL,default=https://api.wanikani.com/v2"`
	AudioStoragePath     string `env:"AUDIO_STORAGE_PATH,default=./audio"`
}
