package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"routekit/orchestrator/internal/model"
	"routekit/orchestrator/internal/payment"
	"routekit/orchestrator/internal/webhook"
)

type Server struct {
	payments *payment.Service
	ingress  *webhook.IngressStore
}

func NewServer(payments *payment.Service, ingress *webhook.IngressStore) *Server {
	return &Server{payments: payments, ingress: ingress}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /charges", s.charge)
	mux.HandleFunc("POST /charges/{id}/capture", s.capture)
	mux.HandleFunc("POST /refunds", s.refund)
	mux.HandleFunc("GET /transactions", s.transactions)
	mux.HandleFunc("POST /webhooks/{psp}", s.webhookIn)
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) charge(w http.ResponseWriter, r *http.Request) {
	var req model.ChargeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	txn, err := s.payments.Charge(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, txn)
}

func (s *Server) capture(w http.ResponseWriter, r *http.Request) {
	var req model.CaptureRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.TransactionID = r.PathValue("id")
	txn, err := s.payments.Capture(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, txn)
}

func (s *Server) refund(w http.ResponseWriter, r *http.Request) {
	var req model.RefundRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	txn, err := s.payments.Refund(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, txn)
}

func (s *Server) transactions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": s.payments.Transactions()})
}

func (s *Server) webhookIn(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	pspCode := strings.ToLower(r.PathValue("psp"))
	eventID := r.Header.Get("X-RouteKit-Sandbox-Event-Id")
	inserted, err := s.ingress.Insert(webhook.InboundEvent{
		PSPCode:           pspCode,
		EventID:           eventID,
		SignatureVerified: r.Header.Get("X-RouteKit-Sandbox-Signature") != "",
		Headers:           flattenHeaders(r.Header),
		RawBody:           body,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"received": true, "deduped": !inserted})
}

func decodeJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dest)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func flattenHeaders(headers http.Header) map[string]string {
	out := map[string]string{}
	for key, value := range headers {
		if len(value) > 0 {
			out[strings.ToLower(key)] = value[0]
		}
	}
	return out
}

