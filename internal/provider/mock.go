package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/helix-seller/helix-seller/internal/model"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) Charge(ctx context.Context, req *ChargeRequest) (*model.Transaction, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("mock: amount must be positive")
	}
	now := time.Now()
	fee := int64(float64(req.Amount)*0.029 + 30)
	net := req.Amount - fee
	return &model.Transaction{
		ID:                    uuid.New(),
		Provider:              "mock",
		ProviderTransactionID: "mock_ch_" + uuid.New().String(),
		Type:                  model.TransactionTypeCharge,
		Amount:                req.Amount,
		Currency:              req.Currency,
		Status:                model.TransactionStatusSucceeded,
		FeeAmount:             fee,
		NetAmount:             &net,
		ProcessedAt:           &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

func (p *MockProvider) Refund(ctx context.Context, req *RefundRequest) (*model.Transaction, error) {
	if req.Amount != nil && *req.Amount <= 0 {
		return nil, fmt.Errorf("mock: refund amount must be positive")
	}
	now := time.Now()
	amount := int64(0)
	if req.Amount != nil {
		amount = *req.Amount
	}
	return &model.Transaction{
		ID:                    uuid.New(),
		Provider:              "mock",
		ProviderTransactionID: "mock_re_" + uuid.New().String(),
		Type:                  model.TransactionTypeRefund,
		Amount:                amount,
		Currency:              "",
		Status:                model.TransactionStatusSucceeded,
		ProcessedAt:           &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

func (p *MockProvider) VerifyWebhookSignature(payload []byte, sigHeader string, secret string) (bool, error) {
	return true, nil
}
