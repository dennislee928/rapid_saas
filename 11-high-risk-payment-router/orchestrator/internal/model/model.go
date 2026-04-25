package model

import "time"

type TransactionState string

const (
	StateCreated           TransactionState = "created"
	StateRouting           TransactionState = "routing"
	StateAuthorising       TransactionState = "authorising"
	StateAuthorised        TransactionState = "authorised"
	StateCaptured          TransactionState = "captured"
	StateRefunded          TransactionState = "refunded"
	StateFailedRetriable   TransactionState = "failed_retriable"
	StateFailedTerminal    TransactionState = "failed_terminal"
	StateRequires3DS       TransactionState = "requires_3ds"
	StatePartiallyRefunded TransactionState = "partially_refunded"
)

type DeclineReason string

const (
	DeclineNone              DeclineReason = ""
	DeclineDoNotHonor        DeclineReason = "do_not_honor"
	DeclineInsufficientFunds DeclineReason = "insufficient_funds"
	DeclineProcessorError    DeclineReason = "processor_error"
	DeclineNetworkError      DeclineReason = "network_error"
	DeclineTimeout           DeclineReason = "timeout"
	DeclineFraud             DeclineReason = "fraud"
)

func (d DeclineReason) Retriable() bool {
	switch d {
	case DeclineDoNotHonor, DeclineProcessorError, DeclineNetworkError, DeclineTimeout:
		return true
	default:
		return false
	}
}

type ChargeRequest struct {
	MerchantID         string `json:"merchant_id"`
	IdempotencyKey     string `json:"-"`
	PaymentMethodToken string `json:"payment_method_token"`
	AmountMinor        int64  `json:"amount_minor"`
	Currency           string `json:"currency"`
	Country            string `json:"country"`
	Brand              string `json:"brand"`
	CardType           string `json:"card_type,omitempty"`
	MCC                string `json:"mcc,omitempty"`
	Capture            bool   `json:"capture"`
}

type CaptureRequest struct {
	TransactionID string `json:"-"`
	MerchantID    string `json:"merchant_id"`
	AmountMinor   int64  `json:"amount_minor,omitempty"`
}

type RefundRequest struct {
	MerchantID     string `json:"merchant_id"`
	TransactionID  string `json:"transaction_id"`
	AmountMinor    int64  `json:"amount_minor"`
	Reason         string `json:"reason,omitempty"`
	IdempotencyKey string `json:"-"`
}

type Transaction struct {
	ID              string           `json:"id"`
	MerchantID      string           `json:"merchant_id"`
	IdempotencyKey  string           `json:"idempotency_key,omitempty"`
	AmountMinor     int64            `json:"amount_minor"`
	Currency        string           `json:"currency"`
	State           TransactionState `json:"state"`
	AttemptCount    int              `json:"attempt_count"`
	CurrentPSP      string           `json:"current_psp,omitempty"`
	CurrentPSPTxnID string           `json:"current_psp_txn_id,omitempty"`
	DeclineReason   DeclineReason    `json:"decline_reason_normalised,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}
