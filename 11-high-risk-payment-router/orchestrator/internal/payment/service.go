package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"routekit/orchestrator/internal/model"
	"routekit/orchestrator/internal/psp"
	"routekit/orchestrator/internal/routing"
	"routekit/orchestrator/internal/webhook"
)

type Service struct {
	mu         sync.Mutex
	engine     *routing.Engine
	psps       map[string]psp.Adapter
	outbox     *webhook.MemoryOutbox
	byID       map[string]model.Transaction
	byIdem     map[string]string
	pspTxnByID map[string]string
}

func NewService(engine *routing.Engine, psps map[string]psp.Adapter, outbox *webhook.MemoryOutbox) *Service {
	return &Service{
		engine:     engine,
		psps:       psps,
		outbox:     outbox,
		byID:       map[string]model.Transaction{},
		byIdem:     map[string]string{},
		pspTxnByID: map[string]string{},
	}
}

func (s *Service) Charge(ctx context.Context, req model.ChargeRequest) (model.Transaction, error) {
	if err := ValidateTokenOnlyCharge(req); err != nil {
		return model.Transaction{}, err
	}

	idemKey := req.MerchantID + ":" + req.IdempotencyKey
	s.mu.Lock()
	if existingID := s.byIdem[idemKey]; existingID != "" {
		existing := s.byID[existingID]
		s.mu.Unlock()
		return existing, nil
	}
	txn := model.Transaction{
		ID:             fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		MerchantID:     req.MerchantID,
		IdempotencyKey: req.IdempotencyKey,
		AmountMinor:    req.AmountMinor,
		Currency:       strings.ToUpper(req.Currency),
		State:          model.StateRouting,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	s.byID[txn.ID] = txn
	s.byIdem[idemKey] = txn.ID
	s.mu.Unlock()

	for _, code := range s.engine.Select(req) {
		adapter, ok := s.psps[code]
		if !ok {
			return model.Transaction{}, psp.ErrUnsupported(code)
		}
		txn.State = model.StateAuthorising
		txn.CurrentPSP = code
		txn.AttemptCount++
		resp, err := adapter.Authorize(ctx, psp.AuthorizeRequest{
			MerchantID:         req.MerchantID,
			PaymentMethodToken: req.PaymentMethodToken,
			AmountMinor:        req.AmountMinor,
			Currency:           req.Currency,
			Country:            req.Country,
			Brand:              req.Brand,
			Capture:            req.Capture,
		})
		if err != nil {
			return model.Transaction{}, err
		}
		txn.CurrentPSPTxnID = resp.PSPTxnID
		txn.DeclineReason = resp.DeclineReason
		if resp.Requires3DS {
			txn.State = model.StateRequires3DS
			return s.persistAndEmit(txn, "transaction.requires_action"), nil
		}
		if resp.Approved {
			if req.Capture {
				txn.State = model.StateCaptured
			} else {
				txn.State = model.StateAuthorised
			}
			return s.persistAndEmit(txn, "transaction.authorised"), nil
		}
		if !resp.DeclineReason.Retriable() {
			txn.State = model.StateFailedTerminal
			return s.persistAndEmit(txn, "transaction.failed"), nil
		}
		txn.State = model.StateFailedRetriable
	}

	txn.State = model.StateFailedTerminal
	return s.persistAndEmit(txn, "transaction.failed"), nil
}

func (s *Service) Capture(ctx context.Context, req model.CaptureRequest) (model.Transaction, error) {
	s.mu.Lock()
	txn, ok := s.byID[req.TransactionID]
	pspTxnID := s.pspTxnByID[req.TransactionID]
	s.mu.Unlock()
	if !ok {
		return model.Transaction{}, errors.New("transaction not found")
	}
	adapter, ok := s.psps[txn.CurrentPSP]
	if !ok {
		return model.Transaction{}, psp.ErrUnsupported(txn.CurrentPSP)
	}
	_, err := adapter.Capture(ctx, psp.CaptureRequest{MerchantID: req.MerchantID, PSPTxnID: pspTxnID, AmountMinor: req.AmountMinor})
	if err != nil {
		return model.Transaction{}, err
	}
	txn.State = model.StateCaptured
	return s.persistAndEmit(txn, "transaction.captured"), nil
}

func (s *Service) Refund(ctx context.Context, req model.RefundRequest) (model.Transaction, error) {
	if req.IdempotencyKey == "" {
		return model.Transaction{}, errors.New("idempotency key is required")
	}
	s.mu.Lock()
	txn, ok := s.byID[req.TransactionID]
	pspTxnID := s.pspTxnByID[req.TransactionID]
	s.mu.Unlock()
	if !ok {
		return model.Transaction{}, errors.New("transaction not found")
	}
	adapter, ok := s.psps[txn.CurrentPSP]
	if !ok {
		return model.Transaction{}, psp.ErrUnsupported(txn.CurrentPSP)
	}
	_, err := adapter.Refund(ctx, psp.RefundRequest{MerchantID: req.MerchantID, PSPTxnID: pspTxnID, AmountMinor: req.AmountMinor, Reason: req.Reason})
	if err != nil {
		return model.Transaction{}, err
	}
	if req.AmountMinor < txn.AmountMinor {
		txn.State = model.StatePartiallyRefunded
	} else {
		txn.State = model.StateRefunded
	}
	return s.persistAndEmit(txn, "transaction.refunded"), nil
}

func (s *Service) Transactions() []model.Transaction {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Transaction, 0, len(s.byID))
	for _, txn := range s.byID {
		out = append(out, txn)
	}
	return out
}

func (s *Service) persistAndEmit(txn model.Transaction, eventType string) model.Transaction {
	txn.UpdatedAt = time.Now()
	s.mu.Lock()
	s.byID[txn.ID] = txn
	if txn.CurrentPSPTxnID != "" {
		s.pspTxnByID[txn.ID] = txn.CurrentPSPTxnID
	}
	s.mu.Unlock()
	s.outbox.Enqueue(webhook.OutboundEvent{
		MerchantID:    txn.MerchantID,
		TransactionID: txn.ID,
		EventType:     eventType,
		Payload:       txn,
	})
	return txn
}
