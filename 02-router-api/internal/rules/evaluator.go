package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
)

type Evaluator interface {
	Evaluate(context.Context, json.RawMessage, map[string]any) (bool, error)
}

type ListResolver interface {
	Contains(context.Context, string, string) (bool, error)
	CIDRs(context.Context, string) ([]string, error)
}

type NoopListResolver struct{}

func (NoopListResolver) Contains(context.Context, string, string) (bool, error) { return false, nil }
func (NoopListResolver) CIDRs(context.Context, string) ([]string, error)        { return nil, nil }

type JSONLogicEvaluator struct {
	lists ListResolver
}

func NewJSONLogicEvaluator(lists ListResolver) *JSONLogicEvaluator {
	return &JSONLogicEvaluator{lists: lists}
}

func (e *JSONLogicEvaluator) Evaluate(ctx context.Context, raw json.RawMessage, data map[string]any) (bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return true, nil
	}
	var expr any
	if err := json.Unmarshal(raw, &expr); err != nil {
		return false, fmt.Errorf("invalid jsonlogic: %w", err)
	}
	value, err := e.eval(ctx, expr, data)
	if err != nil {
		return false, err
	}
	return truthy(value), nil
}

func (e *JSONLogicEvaluator) eval(ctx context.Context, expr any, data map[string]any) (any, error) {
	obj, ok := expr.(map[string]any)
	if !ok || len(obj) != 1 {
		return expr, nil
	}
	for op, rawArgs := range obj {
		args := normalizeArgs(rawArgs)
		switch op {
		case "var":
			if len(args) == 0 {
				return nil, nil
			}
			name, _ := args[0].(string)
			return lookup(data, name), nil
		case "and":
			for _, arg := range args {
				value, err := e.eval(ctx, arg, data)
				if err != nil || !truthy(value) {
					return value, err
				}
			}
			return true, nil
		case "or":
			for _, arg := range args {
				value, err := e.eval(ctx, arg, data)
				if err != nil || truthy(value) {
					return value, err
				}
			}
			return false, nil
		case "==", "!=":
			left, right, err := e.evalPair(ctx, args, data)
			if err != nil {
				return nil, err
			}
			equal := fmt.Sprint(left) == fmt.Sprint(right)
			if op == "!=" {
				return !equal, nil
			}
			return equal, nil
		case ">", "<":
			left, right, err := e.evalPair(ctx, args, data)
			if err != nil {
				return nil, err
			}
			lf, lok := number(left)
			rf, rok := number(right)
			if !lok || !rok {
				return false, nil
			}
			if op == ">" {
				return lf > rf, nil
			}
			return lf < rf, nil
		case "in":
			left, right, err := e.evalPair(ctx, args, data)
			if err != nil {
				return nil, err
			}
			return contains(right, fmt.Sprint(left)), nil
		case "regex_match":
			left, right, err := e.evalPair(ctx, args, data)
			if err != nil {
				return nil, err
			}
			return regexp.MatchString(fmt.Sprint(right), fmt.Sprint(left))
		case "in_list":
			left, right, err := e.evalPair(ctx, args, data)
			if err != nil {
				return nil, err
			}
			return e.lists.Contains(ctx, fmt.Sprint(right), fmt.Sprint(left))
		case "in_cidr":
			left, right, err := e.evalPair(ctx, args, data)
			if err != nil {
				return nil, err
			}
			return e.inCIDR(ctx, fmt.Sprint(left), right)
		default:
			return nil, fmt.Errorf("unsupported jsonlogic operator %q", op)
		}
	}
	return nil, errors.New("empty jsonlogic expression")
}

func (e *JSONLogicEvaluator) evalPair(ctx context.Context, args []any, data map[string]any) (any, any, error) {
	if len(args) < 2 {
		return nil, nil, errors.New("operator requires two arguments")
	}
	left, err := e.eval(ctx, args[0], data)
	if err != nil {
		return nil, nil, err
	}
	right, err := e.eval(ctx, args[1], data)
	return left, right, err
}

func (e *JSONLogicEvaluator) inCIDR(ctx context.Context, ipString string, source any) (bool, error) {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return false, nil
	}
	var cidrs []string
	if ref, ok := source.(map[string]any); ok {
		if listID, ok := ref["list_cidrs"].(string); ok {
			items, err := e.lists.CIDRs(ctx, listID)
			if err != nil {
				return false, err
			}
			cidrs = items
		}
	} else if values, ok := source.([]any); ok {
		for _, value := range values {
			cidrs = append(cidrs, fmt.Sprint(value))
		}
	}
	for _, value := range cidrs {
		_, network, err := net.ParseCIDR(value)
		if err == nil && network.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

func normalizeArgs(raw any) []any {
	if args, ok := raw.([]any); ok {
		return args
	}
	return []any{raw}
}

func lookup(data map[string]any, key string) any {
	if value, ok := data[key]; ok {
		return value
	}
	if body, ok := data["body"].(map[string]any); ok {
		return body[key]
	}
	return nil
}

func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case float64:
		return v != 0
	default:
		return true
	}
}

func number(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

func contains(collection any, needle string) bool {
	switch values := collection.(type) {
	case []any:
		for _, value := range values {
			if fmt.Sprint(value) == needle {
				return true
			}
		}
	case string:
		return regexp.MustCompile(regexp.QuoteMeta(needle)).FindStringIndex(values) != nil
	}
	return false
}
