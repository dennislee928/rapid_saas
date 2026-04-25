package main

import (
	"log"
	"net/http"
	"os"

	"routekit/orchestrator/internal/api"
	"routekit/orchestrator/internal/payment"
	"routekit/orchestrator/internal/psp"
	"routekit/orchestrator/internal/routing"
	"routekit/orchestrator/internal/webhook"
)

func main() {
	addr := getenv("ROUTEKIT_HTTP_ADDR", ":8080")
	psps := psp.DefaultSandboxAdapters()
	engine := routing.NewEngine(routing.DefaultRules())
	service := payment.NewService(engine, psps, webhook.NewMemoryOutbox())
	server := api.NewServer(service, webhook.NewIngressStore())

	log.Printf("routekit orchestrator listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

