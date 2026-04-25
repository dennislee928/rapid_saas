package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"tiltguard/graph-linker/internal/linker"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	service := linker.NewService(linker.Config{SimilarityThreshold: 0.82})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service": "tiltguard-graph-linker"})
	})
	mux.HandleFunc("/queue/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var event linker.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result := service.Link(event)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(result)
	})

	addr := env("ADDR", ":8080")
	log.Info("starting graph linker", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
