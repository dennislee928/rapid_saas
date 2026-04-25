package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/rapid-saas/audio-stem-separation/orchestrator/internal/store"
)

type Config struct {
	SpaceURL string
}

type Server struct {
	store *store.Memory
	cfg   Config
	queue chan store.Job
}

type createJobRequest struct {
	UserID         string         `json:"user_id"`
	IdempotencyKey string         `json:"idempotency_key"`
	Model          string         `json:"model"`
	Params         map[string]any `json:"params"`
	InputR2Key     string         `json:"input_r2_key"`
	InputBytes     int64          `json:"input_bytes"`
	InputSeconds   float64        `json:"input_seconds"`
}

func NewServer(mem *store.Memory, cfg Config) *Server {
	return &Server{store: mem, cfg: cfg, queue: make(chan store.Job, 64)}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /api/jobs", s.createJob)
	mux.HandleFunc("GET /api/jobs/", s.getJob)
	mux.HandleFunc("POST /internal/jobs/", s.internalJobUpdate)
	mux.HandleFunc("POST /internal/stripe/event", s.stripeEvent)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "audio-stem-orchestrator", "queue_depth": len(s.queue)})
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.Model == "" {
		req.Model = "htdemucs"
	}
	if req.InputSeconds <= 0 || req.InputSeconds > 600 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "input_seconds_invalid"})
		return
	}
	if req.InputBytes <= 0 || req.InputBytes > 50*1024*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "input_bytes_invalid"})
		return
	}
	cost := CalculateCostMillicredits(req.InputSeconds, req.Model)
	job, replay, err := s.store.CreateJob(store.Job{
		UserID:         req.UserID,
		IdempotencyKey: req.IdempotencyKey,
		Model:          req.Model,
		Params:         defaultParams(req.Params),
		InputR2Key:     req.InputR2Key,
		InputBytes:     req.InputBytes,
		InputSeconds:   req.InputSeconds,
		CostCredits:    cost,
	})
	if err != nil {
		if errors.Is(err, store.ErrInsufficientCredits) {
			writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": "insufficient_credits"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create_job_failed"})
		return
	}
	if !replay {
		select {
		case s.queue <- job:
		default:
			_, _ = s.store.FailAndRefund(job.ID, "queue_full")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "queue_full"})
			return
		}
	}
	status := http.StatusAccepted
	if replay {
		status = http.StatusOK
	}
	writeJSON(w, status, job)
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	job, ok := s.store.GetJob(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) internalJobUpdate(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/internal/jobs/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	id, action := parts[0], parts[1]
	switch action {
	case "heartbeat":
		if err := s.store.Heartbeat(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "job_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	case "complete":
		var body struct {
			OutputZipKey string `json:"output_zip_key"`
			ErrorCode    string `json:"error_code"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var (
			job store.Job
			err error
		)
		if body.ErrorCode != "" {
			job, err = s.store.FailAndRefund(id, body.ErrorCode)
		} else {
			job, err = s.store.Complete(id, body.OutputZipKey)
		}
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "job_not_found"})
			return
		}
		writeJSON(w, http.StatusOK, job)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
	}
}

func (s *Server) stripeEvent(w http.ResponseWriter, r *http.Request) {
	var event map[string]any
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"received": true, "event_id": event["id"], "mode": "stub"})
}

func (s *Server) DispatchOne(client *http.Client) bool {
	select {
	case job := <-s.queue:
		spaceURL := s.cfg.SpaceURL
		if spaceURL == "" {
			spaceURL = "stub://space"
		}
		_, _ = s.store.MarkDispatched(job.ID, spaceURL)
		if strings.HasPrefix(spaceURL, "stub://") {
			return true
		}
		payload, _ := json.Marshal(map[string]any{
			"job_id":       job.ID,
			"input_url":    "https://r2.local.stub/" + job.InputR2Key,
			"model":        job.Model,
			"params":       job.Params,
			"callback_url": "/internal/jobs/" + job.ID + "/complete",
		})
		resp, err := client.Post(spaceURL+"/infer", "application/json", bytes.NewReader(payload))
		if err != nil || resp.StatusCode >= 500 {
			_, _ = s.store.FailAndRefund(job.ID, "dispatch_failed")
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		return true
	default:
		return false
	}
}

func CalculateCostMillicredits(seconds float64, model string) int {
	multiplier := map[string]float64{
		"htdemucs":     1.0,
		"htdemucs_ft":  2.0,
		"htdemucs_6s":  1.4,
		"mdxnet_vocal": 1.0,
	}[model]
	if multiplier == 0 {
		multiplier = 1.0
	}
	cost := int(seconds * 833 * multiplier)
	if cost < 1 {
		return 1
	}
	return cost
}

func defaultParams(input map[string]any) map[string]any {
	if input == nil {
		input = map[string]any{}
	}
	if _, ok := input["two_stems"]; !ok {
		input["two_stems"] = "vocals"
	}
	if _, ok := input["segment"]; !ok {
		input["segment"] = 7
	}
	if _, ok := input["shifts"]; !ok {
		input["shifts"] = 1
	}
	return input
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
