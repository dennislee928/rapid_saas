package config

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr           string
	WorkerSharedSecret string
	LogLevel           slog.Level
}

func FromEnv() Config {
	return Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		WorkerSharedSecret: os.Getenv("WORKER_SHARED_SECRET"),
		LogLevel:           parseLogLevel(env("LOG_LEVEL", "info")),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
