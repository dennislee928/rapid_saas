package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rapid-saas/router-api/internal/admin"
	"github.com/rapid-saas/router-api/internal/config"
	"github.com/rapid-saas/router-api/internal/queue"
	"github.com/rapid-saas/router-api/internal/store"
)

type Dependencies struct {
	Config     config.Config
	Logger     *slog.Logger
	Admin      *admin.Handler
	Queue      *queue.Handler
	Health     *HealthHandler
	StartedAt  time.Time
	Repository store.HealthRepository
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(accessLog(deps.Logger))

	r.Get("/healthz", deps.Health.Health)
	r.Get("/readyz", deps.Health.Ready)
	r.Post("/_q/consume", deps.Queue.Consume)

	r.Route("/v1", func(r chi.Router) {
		r.Use(admin.TenantMiddleware)
		r.Route("/endpoints", func(r chi.Router) {
			r.Get("/", deps.Admin.ListEndpoints)
			r.Post("/", deps.Admin.CreateEndpoint)
			r.Route("/{endpointID}", func(r chi.Router) {
				r.Get("/", deps.Admin.GetEndpoint)
				r.Put("/", deps.Admin.UpdateEndpoint)
				r.Delete("/", deps.Admin.DeleteEndpoint)
			})
		})
		r.Route("/destinations", func(r chi.Router) {
			r.Get("/", deps.Admin.ListDestinations)
			r.Post("/", deps.Admin.CreateDestination)
			r.Route("/{destinationID}", func(r chi.Router) {
				r.Get("/", deps.Admin.GetDestination)
				r.Put("/", deps.Admin.UpdateDestination)
				r.Delete("/", deps.Admin.DeleteDestination)
			})
		})
		r.Route("/rules", func(r chi.Router) {
			r.Get("/", deps.Admin.ListRules)
			r.Post("/", deps.Admin.CreateRule)
			r.Route("/{ruleID}", func(r chi.Router) {
				r.Get("/", deps.Admin.GetRule)
				r.Put("/", deps.Admin.UpdateRule)
				r.Delete("/", deps.Admin.DeleteRule)
			})
		})
	})

	return r
}

func accessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
