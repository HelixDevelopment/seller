package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/helix-seller/helix-seller/internal/eventbus"
	"github.com/helix-seller/helix-seller/internal/model"
	"github.com/helix-seller/helix-seller/internal/provider"
	"github.com/helix-seller/helix-seller/internal/repository"
)

type PaymentService struct {
	txRepo          *repository.TransactionRepo
	pmRepo          *repository.PaymentMethodRepo
	eventBus        eventbus.EventBus
	logger          *zap.Logger
	idempotencyRepo *repository.IdempotencyRepo
	providerFactory *provider.Factory
}

func NewPaymentService(txRepo *repository.TransactionRepo, pmRepo *repository.PaymentMethodRepo, eventBus eventbus.EventBus, logger *zap.Logger, providerFactory *provider.Factory) *PaymentService {
	var idempotencyRepo *repository.IdempotencyRepo
	if txRepo != nil {
		idempotencyRepo = repository.NewIdempotencyRepo(txRepo.DB())
	}
	return &PaymentService{
		txRepo:          txRepo,
		pmRepo:          pmRepo,
		eventBus:        eventBus,
		logger:          logger,
		idempotencyRepo: idempotencyRepo,
		providerFactory: providerFactory,
	}
}

func (s *PaymentService) ProcessPayment(ctx context.Context, merchantID, customerID, paymentMethodID uuid.UUID, amount int64, currency, idempotencyKey string) (*model.Transaction, error) {
	if idempotencyKey != "" {
		ok, err := s.idempotencyRepo.CheckAndSave(ctx, idempotencyKey, merchantID.String())
		if err != nil {
			s.logger.Error("idempotency check failed", zap.Error(err))
			return nil, fmt.Errorf("idempotency check error: %w", err)
		}
		if !ok {
			s.logger.Info("duplicate request detected via idempotency key",
				zap.String("idempotency_key", idempotencyKey),
			)
			return nil, fmt.Errorf("duplicate idempotency key: %s", idempotencyKey)
		}
	}

	pm, err := s.pmRepo.GetByID(ctx, paymentMethodID)
	if err != nil {
		return nil, err
	}

	tx := &model.Transaction{
		ID:              uuid.New(),
		MerchantID:      merchantID,
		CustomerID:      customerID,
		PaymentMethodID: paymentMethodID,
		Amount:          amount,
		Currency:        currency,
		Status:          model.TransactionStatusPending,
		Type:            model.TransactionTypeCharge,
		Provider:        pm.Provider,
		IdempotencyKey:  idempotencyKey,
	}
	if err := s.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	prov, err := s.providerFactory.Get(pm.Provider)
	if err != nil {
		tx.Status = model.TransactionStatusFailed
		tx.ErrorMessage = err.Error()
		if updateErr := s.txRepo.Update(ctx, tx); updateErr != nil {
			s.logger.Error("failed to persist transaction after provider lookup failure", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("get provider %s: %w", pm.Provider, err)
	}

	chargeResp, err := prov.Charge(ctx, &provider.ChargeRequest{
		Amount:         amount,
		Currency:       currency,
		Source:         pm.ProviderToken,
		Description:    fmt.Sprintf("charge for customer %s", customerID),
		IdempotencyKey: idempotencyKey,
		Metadata: map[string]string{
			"merchant_id": merchantID.String(),
			"customer_id": customerID.String(),
		},
	})
	if err != nil {
		tx.Status = model.TransactionStatusFailed
		tx.ErrorMessage = err.Error()
		if updateErr := s.txRepo.Update(ctx, tx); updateErr != nil {
			s.logger.Error("failed to persist transaction after charge failure", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("provider charge failed: %w", err)
	}

	now := time.Now()
	tx.Provider = chargeResp.Provider
	tx.ProviderTransactionID = chargeResp.ProviderTransactionID
	tx.Status = chargeResp.Status
	tx.FeeAmount = chargeResp.FeeAmount
	tx.NetAmount = chargeResp.NetAmount
	tx.ProcessedAt = &now
	if err := s.txRepo.Update(ctx, tx); err != nil {
		return nil, err
	}

	if err := s.eventBus.Publish(ctx, "events.payment.initiated", &eventbus.Event{
		Type:   "payment.initiated",
		Source: "helix-seller",
		Data:   tx,
	}); err != nil {
		s.logger.Error("failed to publish payment.initiated event", zap.Error(err))
	}

	return tx, nil
}

func (s *PaymentService) GetTransaction(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	return s.txRepo.GetByID(ctx, id)
}

func (s *PaymentService) ListTransactions(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]*model.Transaction, int64, error) {
	return s.txRepo.ListByMerchant(ctx, merchantID, page, pageSize)
}

func (s *PaymentService) Refund(ctx context.Context, transactionID uuid.UUID, amount int64, reason string) (*model.Transaction, error) {
	orig, err := s.txRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	prov, err := s.providerFactory.Get(orig.Provider)
	if err != nil {
		return nil, fmt.Errorf("get provider %s: %w", orig.Provider, err)
	}

	var reqAmount *int64
	if amount > 0 {
		reqAmount = &amount
	}

	refundResp, err := prov.Refund(ctx, &provider.RefundRequest{
		TransactionID: orig.ProviderTransactionID,
		Amount:        reqAmount,
		Reason:        reason,
		Metadata: map[string]string{
			"merchant_id": orig.MerchantID.String(),
			"customer_id": orig.CustomerID.String(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("provider refund failed: %w", err)
	}

	refund := &model.Transaction{
		ID:                    uuid.New(),
		MerchantID:            orig.MerchantID,
		CustomerID:            orig.CustomerID,
		PaymentMethodID:       orig.PaymentMethodID,
		Amount:                amount,
		Currency:              orig.Currency,
		Status:                model.TransactionStatusReversed,
		Type:                  model.TransactionTypeRefund,
		Provider:              refundResp.Provider,
		ProviderTransactionID: refundResp.ProviderTransactionID,
		Description:           reason,
	}
	if err := s.txRepo.Create(ctx, refund); err != nil {
		return nil, err
	}

	if err := s.eventBus.Publish(ctx, "events.payment.refunded", &eventbus.Event{
		Type:   "payment.refunded",
		Source: "helix-seller",
		Data:   refund,
	}); err != nil {
		s.logger.Error("failed to publish payment.refunded event", zap.Error(err))
	}

	return refund, nil
}
