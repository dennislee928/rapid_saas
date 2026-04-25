package main

import (
	"log"
	"net/http"
	"os"

	"github.com/rapid-saas/audio-stem-separation/orchestrator/internal/app"
	"github.com/rapid-saas/audio-stem-separation/orchestrator/internal/store"
)

func main() {
	mem := store.NewMemory()
	mem.SeedUser("user_demo", "demo@example.com", 100000)

	server := app.NewServer(mem, app.Config{
		SpaceURL: os.Getenv("HF_SPACE_URL"),
	})

	addr := ":" + envDefault("PORT", "8080")
	log.Printf("audio stem orchestrator listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Routes()))
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
