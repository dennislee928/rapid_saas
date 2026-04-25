package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/rapid-saas/router-api/internal/delivery"
	"github.com/rapid-saas/router-api/internal/model"
	"github.com/rapid-saas/router-api/internal/retry"
	"github.com/rapid-saas/router-api/internal/rules"
	"github.com/rapid-saas/router-api/internal/store"
)

type Processor struct {
	repo      store.ProcessingRepository
	adminRepo store.AdminRepository
	evaluator rules.Evaluator
	renderer  *rules.TemplateRenderer
	sender    delivery.Sender
	retries   retry.Scheduler
	dlq       retry.DLQ
	logger    *slog.Logger
}

type Result struct {
	EventID string `json:"event_id"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

func NewProcessor(repo store.ProcessingRepository, adminRepo store.AdminRepository, evaluator rules.Evaluator, renderer *rules.TemplateRenderer, sender delivery.Sender, retries retry.Scheduler, dlq retry.DLQ, logger *slog.Logger) *Processor {
	return &Processor{repo: repo, adminRepo: adminRepo, evaluator: evaluator, renderer: renderer, sender: sender, retries: retries, dlq: dlq, logger: logger}
}

func (p *Processor) ProcessBatch(ctx context.Context, events []model.QueueEvent) []Result {
	results := make([]Result, 0, len(events))
	for _, event := range events {
		if err := p.Process(ctx, event); err != nil {
			p.logger.Error("event processing failed", slog.String("event_id", event.ID), slog.Any("error", err))
			results = append(results, Result{EventID: event.ID, Status: "failed", Error: err.Error()})
			continue
		}
		results = append(results, Result{EventID: event.ID, Status: "processed"})
	}
	return results
}

func (p *Processor) Process(ctx context.Context, event model.QueueEvent) error {
	endpoint, err := p.repo.GetEndpoint(ctx, event.TenantID, event.EndpointID)
	if err != nil {
		return err
	}
	if !endpoint.Enabled {
		return p.writeLog(ctx, event, model.Rule{}, model.Destination{}, "dropped", 0, 0, "endpoint_disabled")
	}
	ruleSet, err := p.repo.ListRules(ctx, event.TenantID, event.EndpointID)
	if err != nil {
		return err
	}
	data := rules.EventData(event.Headers, event.Body)
	for _, rule := range ruleSet {
		if !rule.Enabled {
			continue
		}
		matched, err := p.evaluator.Evaluate(ctx, rule.FilterJSONLogic, data)
		if err != nil {
			return p.writeLog(ctx, event, rule, model.Destination{}, "failed", 0, 0, err.Error())
		}
		if !matched {
			continue
		}
		if rule.OnMatch == "drop" {
			return p.writeLog(ctx, event, rule, model.Destination{}, "dropped", 0, 0, "rule_drop")
		}
		destination, err := p.repo.GetDestination(ctx, event.TenantID, rule.DestinationID)
		if err != nil {
			return err
		}
		body, err := p.renderer.Render(rule.TransformKind, rule.TransformBody, data, event.Body)
		if err != nil {
			return p.writeLog(ctx, event, rule, destination, "failed", 0, 0, err.Error())
		}
		result := p.sender.Send(ctx, destination, body)
		if result.Retryable {
			_ = p.retries.Schedule(ctx, retry.Job{Event: event, Rule: rule, Destination: destination, Attempt: 1, DueAt: time.Now().Add(p.retries.Policy().Delay(1))})
		}
		status := "delivered"
		errMessage := ""
		if result.Err != nil {
			status = "failed"
			errMessage = result.Err.Error()
		}
		if err := p.writeLog(ctx, event, rule, destination, status, result.HTTPStatus, result.Latency.Milliseconds(), errMessage); err != nil {
			return err
		}
		if rule.OnMatch != "continue" {
			return nil
		}
	}
	return p.writeLog(ctx, event, model.Rule{}, model.Destination{}, "dropped", 0, 0, "no_rule_matched")
}

func (p *Processor) writeLog(ctx context.Context, event model.QueueEvent, rule model.Rule, destination model.Destination, status string, httpStatus int, latencyMS int64, errMessage string) error {
	return p.repo.WriteDeliveryLog(ctx, model.DeliveryLog{
		TenantID:      event.TenantID,
		EndpointID:    event.EndpointID,
		RuleID:        rule.ID,
		DestinationID: destination.ID,
		Status:        status,
		HTTPStatus:    httpStatus,
		LatencyMS:     latencyMS,
		Error:         errMessage,
		RequestHash:   event.RequestHash,
		RequestSize:   len(event.Body),
		ReceivedAt:    event.ReceivedAt,
		DeliveredAt:   time.Now().UnixMilli(),
	})
}

func DebugEventBody(event model.QueueEvent) map[string]any {
	var out map[string]any
	_ = json.Unmarshal(event.Body, &out)
	return out
}
