package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helix-seller/helix-seller/internal/model"
)

type WebhookDeliveryRepo struct {
	db *pgxpool.Pool
}

func NewWebhookDeliveryRepo(db *pgxpool.Pool) *WebhookDeliveryRepo {
	return &WebhookDeliveryRepo{db: db}
}

func (r *WebhookDeliveryRepo) Create(ctx context.Context, d *model.WebhookDelivery) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, merchant_id, event_type, event_payload, status, attempts, max_attempts, response_code, response_body, last_error, completed_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())`,
		d.ID, d.WebhookID, d.MerchantID, d.EventType, d.EventPayload,
		d.Status, d.Attempts, d.MaxAttempts, d.ResponseCode, d.ResponseBody, d.LastError, d.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert webhook delivery: %w", err)
	}
	return nil
}

func (r *WebhookDeliveryRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.WebhookDeliveryStatus, responseCode int, responseBody, lastError string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE webhook_deliveries
		 SET status = $2, response_code = $3, response_body = $4, last_error = $5,
		     attempts = attempts + 1,
		     completed_at = CASE WHEN $2 IN ('delivered', 'failed') THEN NOW() ELSE completed_at END,
		     updated_at = NOW()
		 WHERE id = $1`,
		id, status, responseCode, responseBody, lastError,
	)
	if err != nil {
		return fmt.Errorf("update webhook delivery status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *WebhookDeliveryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.WebhookDelivery, error) {
	d := &model.WebhookDelivery{}
	err := r.db.QueryRow(ctx,
		`SELECT id, webhook_id, merchant_id, event_type, event_payload, status, attempts, max_attempts, response_code, response_body, last_error, completed_at, created_at, updated_at
		 FROM webhook_deliveries WHERE id = $1`, id,
	).Scan(
		&d.ID, &d.WebhookID, &d.MerchantID, &d.EventType, &d.EventPayload,
		&d.Status, &d.Attempts, &d.MaxAttempts, &d.ResponseCode, &d.ResponseBody,
		&d.LastError, &d.CompletedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query webhook delivery by id: %w", err)
	}
	return d, nil
}

func (r *WebhookDeliveryRepo) ListByMerchant(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*model.WebhookDelivery, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, webhook_id, merchant_id, event_type, event_payload, status, attempts, max_attempts, response_code, response_body, last_error, completed_at, created_at, updated_at
		 FROM webhook_deliveries WHERE merchant_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		merchantID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		d := &model.WebhookDelivery{}
		if err := rows.Scan(
			&d.ID, &d.WebhookID, &d.MerchantID, &d.EventType, &d.EventPayload,
			&d.Status, &d.Attempts, &d.MaxAttempts, &d.ResponseCode, &d.ResponseBody,
			&d.LastError, &d.CompletedAt, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook delivery: %w", err)
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, nil
}


