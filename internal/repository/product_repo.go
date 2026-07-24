package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helix-seller/helix-seller/internal/model"
)

type ProductRepo struct {
	db *pgxpool.Pool
}

func NewProductRepo(db *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) Create(ctx context.Context, p *model.Product) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO products (id, merchant_id, name, description, price, currency, status, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		p.ID, p.MerchantID, p.Name, p.Description, p.Price, p.Currency, p.Status, p.Metadata,
	)
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id, merchantID string) (*model.Product, error) {
	p := &model.Product{}
	err := r.db.QueryRow(ctx,
		`SELECT id, merchant_id, name, description, price, currency, status, metadata, created_at, updated_at, deleted_at
		 FROM products WHERE id = $1 AND merchant_id = $2 AND deleted_at IS NULL`, id, merchantID,
	).Scan(&p.ID, &p.MerchantID, &p.Name, &p.Description, &p.Price, &p.Currency, &p.Status, &p.Metadata, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if err == pgx.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query product by id: %w", err)
	}
	return p, nil
}

func (r *ProductRepo) ListByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*model.Product, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM products WHERE merchant_id = $1 AND deleted_at IS NULL`, merchantID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, merchant_id, name, description, price, currency, status, metadata, created_at, updated_at, deleted_at
		 FROM products WHERE merchant_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		merchantID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p := &model.Product{}
		if err := rows.Scan(&p.ID, &p.MerchantID, &p.Name, &p.Description, &p.Price, &p.Currency, &p.Status, &p.Metadata, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, total, nil
}

func (r *ProductRepo) Update(ctx context.Context, p *model.Product) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE products SET name=$3, description=$4, price=$5, currency=$6, status=$7, metadata=$8, updated_at=NOW()
		 WHERE id=$1 AND merchant_id=$2 AND deleted_at IS NULL`,
		p.ID, p.MerchantID, p.Name, p.Description, p.Price, p.Currency, p.Status, p.Metadata,
	)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *ProductRepo) Delete(ctx context.Context, id, merchantID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE products SET deleted_at=$3 WHERE id=$1 AND merchant_id=$2 AND deleted_at IS NULL`,
		id, merchantID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
