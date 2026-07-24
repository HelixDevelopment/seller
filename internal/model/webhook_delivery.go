package model

import (
	"time"

	"github.com/google/uuid"
)

type WebhookDeliveryStatus string

const (
	WebhookDeliveryPending   WebhookDeliveryStatus = "pending"
	WebhookDeliveryDelivered WebhookDeliveryStatus = "delivered"
	WebhookDeliveryFailed    WebhookDeliveryStatus = "failed"
	WebhookDeliveryRetrying  WebhookDeliveryStatus = "retrying"
)

type WebhookDelivery struct {
	ID           uuid.UUID             `json:"id"`
	WebhookID    uuid.UUID             `json:"webhook_id"`
	MerchantID   uuid.UUID             `json:"merchant_id"`
	EventType    string                `json:"event_type"`
	EventPayload string                `json:"event_payload"`
	Status       WebhookDeliveryStatus `json:"status"`
	Attempts     int                   `json:"attempts"`
	MaxAttempts  int                   `json:"max_attempts"`
	ResponseCode int                   `json:"response_code"`
	ResponseBody string                `json:"response_body"`
	LastError    string                `json:"last_error"`
	CompletedAt  *time.Time            `json:"completed_at,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}
