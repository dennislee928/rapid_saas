package payment

import (
	"errors"
	"regexp"

	"routekit/orchestrator/internal/model"
)

var rawPANPattern = regexp.MustCompile(`\b[0-9][0-9 -]{11,22}[0-9]\b`)

func ValidateTokenOnlyCharge(req model.ChargeRequest) error {
	if req.MerchantID == "" {
		return errors.New("merchant_id is required")
	}
	if req.IdempotencyKey == "" {
		return errors.New("idempotency key is required")
	}
	if req.PaymentMethodToken == "" {
		return errors.New("payment_method_token is required")
	}
	if looksLikeRawPAN(req.PaymentMethodToken) {
		return errors.New("raw card data is not accepted; send a vault token")
	}
	if req.AmountMinor <= 0 {
		return errors.New("amount_minor must be positive")
	}
	if len(req.Currency) != 3 {
		return errors.New("currency must be ISO-4217 alpha-3")
	}
	return nil
}

func looksLikeRawPAN(value string) bool {
	return rawPANPattern.MatchString(value)
}
