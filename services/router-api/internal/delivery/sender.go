package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/rapid-saas/router-api/internal/model"
)

type Sender interface {
	Send(context.Context, model.Destination, []byte) Result
}

type Result struct {
	HTTPStatus int
	Latency    time.Duration
	Retryable  bool
	Err        error
}

type HTTPSender struct {
	logger *slog.Logger
	client *http.Client
}

func NewHTTPSender(logger *slog.Logger, client *http.Client) *HTTPSender {
	return &HTTPSender{logger: logger, client: client}
}

func (s *HTTPSender) Send(ctx context.Context, destination model.Destination, body []byte) Result {
	var cfg model.DestinationConfig
	if err := json.Unmarshal(destination.ConfigJSON, &cfg); err != nil {
		return Result{Err: err}
	}
	if cfg.URL == "" {
		return Result{Err: errors.New("destination URL is required")}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return Result{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}
	start := time.Now()
	resp, err := s.client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return Result{Latency: latency, Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		return Result{HTTPStatus: resp.StatusCode, Latency: latency, Retryable: true, Err: errors.New(resp.Status)}
	}
	if resp.StatusCode >= 400 {
		return Result{HTTPStatus: resp.StatusCode, Latency: latency, Err: errors.New(resp.Status)}
	}
	return Result{HTTPStatus: resp.StatusCode, Latency: latency}
}
