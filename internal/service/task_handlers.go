package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/helix-seller/helix-seller/internal/model"
	"github.com/helix-seller/helix-seller/internal/repository"
)

type payoutTaskHandler struct {
	payoutRepo *repository.PayoutRepo
	logger     *zap.Logger
}

func NewPayoutTaskHandler(payoutRepo *repository.PayoutRepo, logger *zap.Logger) TaskHandler {
	return &payoutTaskHandler{payoutRepo: payoutRepo, logger: logger}
}

func (h *payoutTaskHandler) Type() string { return "payout" }

func (h *payoutTaskHandler) HandleTask(ctx context.Context, payload []byte) error {
	var req struct {
		PayoutID string `json:"payout_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("payout task: unmarshal payload: %w", err)
	}
	h.logger.Info("processing payout task", zap.String("payout_id", req.PayoutID))
	id, err := uuid.Parse(req.PayoutID)
	if err != nil {
		return fmt.Errorf("payout task: parse payout_id: %w", err)
	}
	if err := h.payoutRepo.UpdateStatus(ctx, id, model.PayoutStatusInTransit); err != nil {
		return fmt.Errorf("payout task: update status: %w", err)
	}
	return nil
}

type reconciliationTaskHandler struct {
	txRepo *repository.TransactionRepo
	logger *zap.Logger
}

func NewReconciliationTaskHandler(txRepo *repository.TransactionRepo, logger *zap.Logger) TaskHandler {
	return &reconciliationTaskHandler{txRepo: txRepo, logger: logger}
}

func (h *reconciliationTaskHandler) Type() string { return "reconciliation" }

func (h *reconciliationTaskHandler) HandleTask(ctx context.Context, payload []byte) error {
	var req struct {
		MerchantID string `json:"merchant_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("reconciliation task: unmarshal payload: %w", err)
	}
	h.logger.Info("starting reconciliation", zap.String("merchant_id", req.MerchantID))
	merchantID, err := uuid.Parse(req.MerchantID)
	if err != nil {
		return fmt.Errorf("reconciliation task: parse merchant_id: %w", err)
	}
	_, total, err := h.txRepo.ListByMerchant(ctx, merchantID, 1, 1)
	if err != nil {
		return fmt.Errorf("reconciliation task: list transactions: %w", err)
	}
	h.logger.Info("reconciliation complete",
		zap.String("merchant_id", req.MerchantID),
		zap.Int64("total_transactions", total),
	)
	return nil
}

type invoiceTaskHandler struct {
	invoiceRepo *repository.InvoiceRepo
	logger      *zap.Logger
}

func NewInvoiceTaskHandler(invoiceRepo *repository.InvoiceRepo, logger *zap.Logger) TaskHandler {
	return &invoiceTaskHandler{invoiceRepo: invoiceRepo, logger: logger}
}

func (h *invoiceTaskHandler) Type() string { return "invoice" }

func (h *invoiceTaskHandler) HandleTask(ctx context.Context, payload []byte) error {
	var req struct {
		InvoiceID string `json:"invoice_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invoice task: unmarshal payload: %w", err)
	}
	h.logger.Info("processing invoice task", zap.String("invoice_id", req.InvoiceID))
	id, err := uuid.Parse(req.InvoiceID)
	if err != nil {
		return fmt.Errorf("invoice task: parse invoice_id: %w", err)
	}
	if err := h.invoiceRepo.UpdateStatus(ctx, id, model.InvoiceStatusOpen); err != nil {
		return fmt.Errorf("invoice task: update status: %w", err)
	}
	return nil
}

type webhookDeliveryTaskHandler struct {
	webhookConfigRepo *repository.WebhookConfigRepo
	logger            *zap.Logger
}

func NewWebhookDeliveryTaskHandler(webhookConfigRepo *repository.WebhookConfigRepo, logger *zap.Logger) TaskHandler {
	return &webhookDeliveryTaskHandler{webhookConfigRepo: webhookConfigRepo, logger: logger}
}

func (h *webhookDeliveryTaskHandler) Type() string { return "webhook_delivery" }

func (h *webhookDeliveryTaskHandler) HandleTask(ctx context.Context, payload []byte) error {
	var req struct {
		WebhookConfigID string          `json:"webhook_config_id"`
		EventType       string          `json:"event_type"`
		EventPayload    json.RawMessage `json:"event_payload"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("webhook delivery task: unmarshal payload: %w", err)
	}
	h.logger.Info("processing webhook delivery task",
		zap.String("webhook_config_id", req.WebhookConfigID),
		zap.String("event_type", req.EventType),
	)
	id, err := uuid.Parse(req.WebhookConfigID)
	if err != nil {
		return fmt.Errorf("webhook delivery task: parse webhook_config_id: %w", err)
	}
	config, err := h.webhookConfigRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("webhook delivery task: get config: %w", err)
	}
	h.logger.Info("would POST to URL with HMAC signature",
		zap.String("url", config.URL),
		zap.String("event_type", req.EventType),
	)
	return nil
}
