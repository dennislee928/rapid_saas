package admin

import (
	"context"
	"net/http"
)

type contextKey string

const tenantContextKey contextKey = "tenant_id"

func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			writeError(w, http.StatusUnauthorized, "missing X-Tenant-ID")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tenantContextKey, tenantID)))
	})
}

func TenantID(ctx context.Context) string {
	value, _ := ctx.Value(tenantContextKey).(string)
	return value
}
