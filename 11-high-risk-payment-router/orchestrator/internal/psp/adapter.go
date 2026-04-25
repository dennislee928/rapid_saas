package psp

import (
	"context"
	"fmt"
	"time"

	"routekit/orchestrator/internal/model"
)

type Adapter interface {
	Code() string
	Authorize(ctx context.Context, req AuthorizeRequest) (AuthorizeResponse, error)
	Capture(ctx context.Context, req CaptureRequest) (CaptureResponse, error)
	Refund(ctx context.Context, req RefundRequest) (RefundResponse, error)
	VerifyWebhook(headers map[string]string, body []byte) (WebhookEvent, error)
}

type AuthorizeRequest struct {
	MerchantID         string
	PaymentMethodToken string
	AmountMinor        int64
	Currency           string
	Country            string
	Brand              string
	Capture            bool
}

type AuthorizeResponse struct {
	PSPTxnID      string
	Approved      bool
	Requires3DS   bool
	RawCode       string
	DeclineReason model.DeclineReason
	Latency       time.Duration
}

type CaptureRequest struct {
	MerchantID   string
	PSPTxnID     string
	AmountMinor  int64
}

type CaptureResponse struct {
	PSPTxnID string
	Captured bool
}

type RefundRequest struct {
	MerchantID  string
	PSPTxnID    string
	AmountMinor int64
	Reason      string
}

type RefundResponse struct {
	PSPRefundID string
	Refunded    bool
}

type WebhookEvent struct {
	EventID       string
	PSPCode       string
	TransactionID string
	Type          string
}

func DefaultSandboxAdapters() map[string]Adapter {
	adapters := []Adapter{
		NewSandbox("nuvei"),
		NewSandbox("trust"),
		NewSandbox("worldpay"),
		NewSandbox("mollie"),
	}
	result := make(map[string]Adapter, len(adapters))
	for _, adapter := range adapters {
		result[adapter.Code()] = adapter
	}
	return result
}

func ErrUnsupported(code string) error {
	return fmt.Errorf("psp adapter %q is not configured", code)
}

