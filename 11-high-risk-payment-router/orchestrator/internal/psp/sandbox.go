package psp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"routekit/orchestrator/internal/model"
)

type Sandbox struct {
	code string
}

func NewSandbox(code string) *Sandbox {
	return &Sandbox{code: code}
}

func (s *Sandbox) Code() string {
	return s.code
}

func (s *Sandbox) Authorize(ctx context.Context, req AuthorizeRequest) (AuthorizeResponse, error) {
	start := time.Now()
	if err := ctx.Err(); err != nil {
		return AuthorizeResponse{}, err
	}
	if req.PaymentMethodToken == "" {
		return AuthorizeResponse{}, errors.New("payment method token is required")
	}

	switch {
	case strings.Contains(req.PaymentMethodToken, "decline_insufficient"):
		return declined(start, "51", model.DeclineInsufficientFunds), nil
	case strings.Contains(req.PaymentMethodToken, "decline_do_not_honor"):
		return declined(start, "05", model.DeclineDoNotHonor), nil
	case strings.Contains(req.PaymentMethodToken, "processor_error"):
		return declined(start, "96", model.DeclineProcessorError), nil
	case strings.Contains(req.PaymentMethodToken, "requires_3ds"):
		return AuthorizeResponse{
			PSPTxnID:    fmt.Sprintf("%s_3ds_%d", s.code, time.Now().UnixNano()),
			Approved:    false,
			Requires3DS: true,
			RawCode:     "3DS_REQUIRED",
			Latency:     time.Since(start),
		}, nil
	default:
		return AuthorizeResponse{
			PSPTxnID:  fmt.Sprintf("%s_auth_%d", s.code, time.Now().UnixNano()),
			Approved:  true,
			RawCode:   "00",
			Latency:   time.Since(start),
		}, nil
	}
}

func (s *Sandbox) Capture(ctx context.Context, req CaptureRequest) (CaptureResponse, error) {
	if err := ctx.Err(); err != nil {
		return CaptureResponse{}, err
	}
	if req.PSPTxnID == "" {
		return CaptureResponse{}, errors.New("psp transaction id is required")
	}
	return CaptureResponse{PSPTxnID: req.PSPTxnID, Captured: true}, nil
}

func (s *Sandbox) Refund(ctx context.Context, req RefundRequest) (RefundResponse, error) {
	if err := ctx.Err(); err != nil {
		return RefundResponse{}, err
	}
	if req.PSPTxnID == "" {
		return RefundResponse{}, errors.New("psp transaction id is required")
	}
	return RefundResponse{PSPRefundID: fmt.Sprintf("%s_refund_%d", s.code, time.Now().UnixNano()), Refunded: true}, nil
}

func (s *Sandbox) VerifyWebhook(headers map[string]string, body []byte) (WebhookEvent, error) {
	eventID := headers["x-routekit-sandbox-event-id"]
	if eventID == "" {
		return WebhookEvent{}, errors.New("missing sandbox event id")
	}
	secret := "sandbox_" + s.code + "_secret"
	expected := sign(secret, string(body))
	if !hmac.Equal([]byte(expected), []byte(headers["x-routekit-sandbox-signature"])) {
		return WebhookEvent{}, errors.New("invalid sandbox webhook signature")
	}
	return WebhookEvent{EventID: eventID, PSPCode: s.code, Type: "payment.updated"}, nil
}

func declined(start time.Time, raw string, reason model.DeclineReason) AuthorizeResponse {
	return AuthorizeResponse{
		Approved:      false,
		RawCode:       raw,
		DeclineReason: reason,
		Latency:       time.Since(start),
	}
}

func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

