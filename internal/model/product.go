package model

import (
	"encoding/json"
	"time"
)

type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
	ProductStatusArchived ProductStatus = "archived"
)

type Product struct {
	ID          string          `json:"id" db:"id"`
	MerchantID  string          `json:"merchant_id" db:"merchant_id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Price       int64           `json:"price" db:"price"`
	Currency    string          `json:"currency" db:"currency"`
	Status      ProductStatus   `json:"status" db:"status"`
	Metadata    json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}
