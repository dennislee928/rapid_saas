package routing

import (
	"strings"

	"routekit/orchestrator/internal/model"
)

type Rule struct {
	ID         string
	Priority   int
	Countries  []string
	Currencies []string
	Brands     []string
	AmountLTE  int64
	PSPs       []string
	Enabled    bool
}

type Engine struct {
	rules []Rule
}

func NewEngine(rules []Rule) *Engine {
	return &Engine{rules: rules}
}

func DefaultRules() []Rule {
	return []Rule{
		{ID: "uk-visa-nuvei", Priority: 10, Countries: []string{"GB", "UK"}, Brands: []string{"visa"}, AmountLTE: 20000, PSPs: []string{"nuvei", "worldpay", "trust"}, Enabled: true},
		{ID: "uk-mastercard-worldpay", Priority: 20, Countries: []string{"GB", "UK"}, Brands: []string{"mastercard"}, PSPs: []string{"worldpay", "trust", "nuvei"}, Enabled: true},
		{ID: "eu-trust", Priority: 30, Countries: []string{"DE", "FR", "NL", "IE"}, PSPs: []string{"trust", "nuvei", "worldpay"}, Enabled: true},
		{ID: "fallback-pool", Priority: 1000, PSPs: []string{"nuvei", "worldpay", "trust", "mollie"}, Enabled: true},
	}
}

func (e *Engine) Select(req model.ChargeRequest) []string {
	for _, rule := range e.rules {
		if rule.Enabled && rule.matches(req) {
			return append([]string(nil), rule.PSPs...)
		}
	}
	return []string{"nuvei", "worldpay", "trust"}
}

func (r Rule) matches(req model.ChargeRequest) bool {
	if len(r.Countries) > 0 && !containsFold(r.Countries, req.Country) {
		return false
	}
	if len(r.Currencies) > 0 && !containsFold(r.Currencies, req.Currency) {
		return false
	}
	if len(r.Brands) > 0 && !containsFold(r.Brands, req.Brand) {
		return false
	}
	if r.AmountLTE > 0 && req.AmountMinor > r.AmountLTE {
		return false
	}
	return true
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}

