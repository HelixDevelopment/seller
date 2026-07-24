package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyRepo struct {
	db *pgxpool.Pool
}

func NewIdempotencyRepo(db *pgxpool.Pool) *IdempotencyRepo {
	return &IdempotencyRepo{db: db}
}

func (r *IdempotencyRepo) CheckAndSave(ctx context.Context, key, merchantID string) (bool, error) {
	tag, err := r.db.Exec(ctx,
		`INSERT INTO idempotency_keys (key_hash, response, status_code, merchant_id, expires_at)
		 VALUES ($1, '{}'::jsonb, 0, $2, NOW() + INTERVAL '24 hours')
		 ON CONFLICT (key_hash) DO NOTHING`,
		key, merchantID,
	)
	if err != nil {
		return false, fmt.Errorf("idempotency: save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, nil
}
