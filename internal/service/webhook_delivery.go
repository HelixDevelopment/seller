package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/helix-seller/helix-seller/internal/model"
	"github.com/helix-seller/helix-seller/internal/repository"
)

type WebhookDeliveryService struct {
	repo   *repository.WebhookDeliveryRepo
	logger *zap.Logger
}

func NewWebhookDeliveryService(repo *repository.WebhookDeliveryRepo, logger *zap.Logger) *WebhookDeliveryService {
	return &WebhookDeliveryService{repo: repo, logger: logger}
}

func (s *WebhookDeliveryService) ListDeliveries(ctx context.Context, merchantID uuid.UUID, page, pageSize int) ([]*model.WebhookDelivery, int, error) {
	offset := (page - 1) * pageSize
	deliveries, total, err := s.repo.ListByMerchant(ctx, merchantID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	return deliveries, total, nil
}

func (s *WebhookDeliveryService) GetDelivery(ctx context.Context, id uuid.UUID) (*model.WebhookDelivery, error) {
	return s.repo.GetByID(ctx, id)
}
