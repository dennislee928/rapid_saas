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

	"github.com/rapid-saas/router-api/internal/admin"
	"github.com/rapid-saas/router-api/internal/app"
	"github.com/rapid-saas/router-api/internal/config"
	"github.com/rapid-saas/router-api/internal/delivery"
	"github.com/rapid-saas/router-api/internal/queue"
	"github.com/rapid-saas/router-api/internal/retry"
	"github.com/rapid-saas/router-api/internal/rules"
	"github.com/rapid-saas/router-api/internal/store"
)

func main() {
	cfg := config.FromEnv()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	repo := store.NewMemoryStore()
	sender := delivery.NewHTTPSender(logger, delivery.DefaultHTTPClient())
	renderer := rules.NewTemplateRenderer()
	evaluator := rules.NewJSONLogicEvaluator(rules.NoopListResolver{})
	retries := retry.NewMemoryScheduler(retry.DefaultPolicy())
	dlq := retry.NewMemoryDLQ()

	processor := queue.NewProcessor(repo, repo, evaluator, renderer, sender, retries, dlq, logger)
	router := app.NewRouter(app.Dependencies{
		Config:     cfg,
		Logger:     logger,
		Admin:      admin.NewHandler(repo, logger),
		Queue:      queue.NewHandler(processor, logger, cfg.WorkerSharedSecret),
		Health:     app.NewHealthHandler(logger),
		StartedAt:  time.Now(),
		Repository: repo,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("router-api listening", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("router-api stopped")
}
