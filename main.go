package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/0x5d/shiru/internal/api"
	"github.com/0x5d/shiru/internal/config"
	"github.com/0x5d/shiru/internal/postgres"
	"github.com/go-logr/stdr"
	"github.com/sethvargo/go-envconfig"
)

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

	srv := api.NewServer(logger, settingsRepo, vocabRepo)

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
}
