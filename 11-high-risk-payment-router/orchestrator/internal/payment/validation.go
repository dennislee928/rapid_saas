package payment

import (
	"errors"
	"regexp"
	"strings"

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
	if !rawPANPattern.MatchString(value) {
		return false
	}
	digits := strings.NewReplacer(" ", "", "-", "").Replace(value)
	return len(digits) >= 13 && len(digits) <= 19 && luhn(digits)
}

func luhn(digits string) bool {
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if n < 0 || n > 9 {
			return false
		}
		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return sum%10 == 0
}
