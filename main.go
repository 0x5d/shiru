package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	shiruanthropic "github.com/0x5d/shiru/internal/anthropic"
	"github.com/0x5d/shiru/internal/api"
	"github.com/0x5d/shiru/internal/audio"
	"github.com/0x5d/shiru/internal/config"
	"github.com/0x5d/shiru/internal/dictionary"
	"github.com/0x5d/shiru/internal/elasticsearch"
	"github.com/0x5d/shiru/internal/elevenlabs"
	"github.com/0x5d/shiru/internal/postgres"
	"github.com/0x5d/shiru/internal/story"
	"github.com/0x5d/shiru/internal/wanikani"
	"github.com/go-logr/stdr"
	"github.com/sethvargo/go-envconfig"
)

var _ story.Indexer = (*storyIndexAdapter)(nil)

type storyIndexAdapter struct {
	es elasticsearch.Client
}

func (a *storyIndexAdapter) Index(ctx context.Context, s *story.Story) error {
	return a.es.IndexStory(ctx, elasticsearch.StoryDocument{
		StoryID:   s.ID.String(),
		UserID:    s.UserID.String(),
		Topic:     s.Topic,
		Tone:      s.Tone,
		Content:   s.Content,
		JLPTLevel: s.JLPTLevel,
		CreatedAt: s.CreatedAt,
	})
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger := stdr.New(log.Default())

	var cfg config.Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		logger.Error(err, "loading config")
		os.Exit(1)
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error(err, "connecting to database")
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		logger.Error(err, "running migrations")
		os.Exit(1)
	}

	settingsRepo := postgres.NewSettingsRepository(pool)
	vocabRepo := postgres.NewVocabRepository(pool)
	storyRepo := story.NewPostgresRepository(pool)
	audioRepo := audio.NewPostgresRepository(pool)
	tagRepo := postgres.NewTagRepository(pool)
	topicSnapshotRepo := postgres.NewTopicSnapshotRepository(pool)

	esClient := elasticsearch.New(cfg.ElasticsearchURL)
	if err := esClient.EnsureIndex(ctx); err != nil {
		logger.Error(err, "ensuring elasticsearch index")
		os.Exit(1)
	}

	dictClient := dictionary.New(cfg.DictionaryAPIBaseURL)
	anthropicClient := shiruanthropic.New(cfg.AnthropicAPIKey, cfg.AnthropicModel)

	var elClient elevenlabs.Client
	if cfg.ElevenLabsAPIKey != "" && cfg.ElevenLabsVoiceID != "" {
		elClient = elevenlabs.New(cfg.ElevenLabsAPIKey, cfg.ElevenLabsVoiceID)
	}

	wkClient := wanikani.New(cfg.WaniKaniAPIBaseURL)
	audioStore := audio.NewS3FileStore(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3UseSSL)

	indexer := &storyIndexAdapter{es: esClient}
	storySvc := story.NewService(anthropicClient, storyRepo, tagRepo, indexer, logger)

	srv := api.NewServer(
		ctx, logger, settingsRepo, vocabRepo, storyRepo, storySvc, tagRepo, anthropicClient, esClient, dictClient,
		elClient, wkClient, audioRepo, audioStore, topicSnapshotRepo, cfg.ElevenLabsVoiceID,
	)

	httpSrv := &http.Server{
		Addr:    ":8080",
		Handler: srv,
	}

	go func() {
		<-ctx.Done()
		_ = httpSrv.Shutdown(context.Background()) // nosemgrep: go.lang.security.audit.net.use-tls.use-tls
	}()

	logger.Info("starting server", "addr", ":8080")
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed { // nosemgrep: go.lang.security.audit.net.use-tls.use-tls
		logger.Error(err, "server error")
		os.Exit(1)
	}

	logger.Info("waiting for background tasks to complete")
	srv.WaitForBackground()
}
