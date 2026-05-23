package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/echo"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/enrichment"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/httpapi"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/migrations"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/pipeline"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/repository"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/storage"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/transcription"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.FromEnv()
	if err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database pool failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	if err := migrations.Up(ctx, cfg.DatabaseURL); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	repo := repository.New(pool)
	if err := repo.RecoverStuckRows(ctx); err != nil {
		logger.Error("startup recovery failed", "error", err)
		os.Exit(1)
	}

	echoRepo := repository.NewEchoRepository(pool)
	if err := echoRepo.RecoverStuckEchoes(ctx); err != nil {
		logger.Error("echo startup recovery failed", "error", err)
		os.Exit(1)
	}

	blobStore, err := storage.NewS3Store(ctx, cfg.S3)
	if err != nil {
		logger.Error("s3 setup failed", "error", err)
		os.Exit(1)
	}

	if cfg.WorkerEnabled {
		worker := pipeline.New(
			repo,
			transcription.NewOpenAITranscriber(cfg.OpenAIAPIKey, blobStore),
			enrichment.NewOpenAIEnricher(cfg.OpenAIAPIKey),
			logger,
		)
		go worker.Run(ctx)

		echoWorker := pipeline.NewEchoPipeline(
			echoRepo,
			echo.NewOpenAIGenerator(cfg.OpenAIAPIKey),
			logger,
			[]domain.Category{
				domain.CategoryFeeling,
				domain.CategoryIdea,
				domain.CategoryObservation,
				domain.CategoryLearning,
			},
			string(echo.Model),
			echo.PromptVersion,
		)
		go echoWorker.Run(ctx)
	}

	router := httpapi.NewRouter(cfg, logger, repo, echoRepo, blobStore, user.SeededResolver{})
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api started", "port", cfg.Port, "env", cfg.AppEnv, "database_host", cfg.DatabaseHost())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
}
