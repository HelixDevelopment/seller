package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/helix-seller/helix-seller/internal/model"
	"github.com/helix-seller/helix-seller/internal/repository"
)

type WebhookService struct {
	configRepo   *repository.WebhookConfigRepo
	deliveryRepo *repository.WebhookDeliveryRepo
	logger       *zap.Logger
	client       *http.Client
}

func NewWebhookService(configRepo *repository.WebhookConfigRepo, deliveryRepo *repository.WebhookDeliveryRepo, logger *zap.Logger) *WebhookService {
	return &WebhookService{
		configRepo:   configRepo,
		deliveryRepo: deliveryRepo,
		logger:       logger,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *WebhookService) CreateWebhook(ctx context.Context, w *model.WebhookConfig) error {
	w.ID = uuid.New()
	w.CreatedAt = time.Now()
	w.UpdatedAt = time.Now()
	return s.configRepo.Create(ctx, w)
}

func (s *WebhookService) GetWebhook(ctx context.Context, id uuid.UUID) (*model.WebhookConfig, error) {
	return s.configRepo.GetByID(ctx, id)
}

func (s *WebhookService) ListWebhooks(ctx context.Context, merchantID uuid.UUID) ([]*model.WebhookConfig, error) {
	return s.configRepo.ListByMerchant(ctx, merchantID)
}

func (s *WebhookService) UpdateWebhook(ctx context.Context, w *model.WebhookConfig) error {
	w.UpdatedAt = time.Now()
	return s.configRepo.Update(ctx, w)
}

func (s *WebhookService) DeleteWebhook(ctx context.Context, id uuid.UUID) error {
	return s.configRepo.Delete(ctx, id)
}

func (s *WebhookService) Deliver(ctx context.Context, merchantID uuid.UUID, eventType string, payload interface{}) error {
	configs, err := s.configRepo.ListByMerchant(ctx, merchantID)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(map[string]interface{}{
		"id":        uuid.New().String(),
		"type":      eventType,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      payload,
	})

	now := time.Now()
	for _, config := range configs {
		if config.IsActive && s.eventMatches(config.Events, eventType) {
			delivery := &model.WebhookDelivery{
				ID:           uuid.New(),
				WebhookID:    config.ID,
				MerchantID:   merchantID,
				EventType:    eventType,
				EventPayload: string(body),
				Status:       model.WebhookDeliveryPending,
				MaxAttempts:  5,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := s.deliveryRepo.Create(ctx, delivery); err != nil {
				s.logger.Error("failed to create delivery record",
					zap.String("webhook_id", config.ID.String()),
					zap.Error(err),
				)
				continue
			}
			go s.sendWithRetry(delivery.ID, config, body)
		}
	}
	return nil
}

func (s *WebhookService) eventMatches(events []string, eventType string) bool {
	for _, e := range events {
		if e == "*" || e == eventType {
			return true
		}
	}
	return false
}

func (s *WebhookService) sendWithRetry(deliveryID uuid.UUID, config *model.WebhookConfig, body []byte) {
	ctx := context.Background()
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		code, respBody, err := s.send(config, body)
		if err == nil {
			if s.deliveryRepo != nil {
				s.deliveryRepo.UpdateStatus(ctx, deliveryID, model.WebhookDeliveryDelivered, code, respBody, "")
			}
			return
		}
		if s.deliveryRepo != nil {
			status := model.WebhookDeliveryRetrying
			if i == maxRetries-1 {
				status = model.WebhookDeliveryFailed
			}
			s.deliveryRepo.UpdateStatus(ctx, deliveryID, status, code, respBody, err.Error())
		}
		if i < maxRetries-1 {
			delay := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(delay)
		}
	}
	s.logger.Error("webhook delivery failed after retries",
		zap.String("webhook_id", config.ID.String()),
		zap.String("url", config.URL),
	)
}

func (s *WebhookService) send(config *model.WebhookConfig, body []byte) (int, string, error) {
	req, err := http.NewRequest("POST", config.URL, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Helix-Webhook-ID", config.ID.String())

	if config.Secret != "" {
		mac := hmac.New(sha256.New, []byte(config.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Helix-Signature", sig)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	respBody := new(strings.Builder)
	io.Copy(respBody, resp.Body)
	bodyStr := respBody.String()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp.StatusCode, bodyStr, nil
	}
	return resp.StatusCode, bodyStr, fmt.Errorf("webhook returned status %d", resp.StatusCode)
}
