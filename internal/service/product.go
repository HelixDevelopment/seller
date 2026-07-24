package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/helix-seller/helix-seller/internal/model"
	"github.com/helix-seller/helix-seller/internal/repository"
)

type ProductService struct {
	productRepo *repository.ProductRepo
	logger      *zap.Logger
}

func NewProductService(productRepo *repository.ProductRepo, logger *zap.Logger) *ProductService {
	return &ProductService{productRepo: productRepo, logger: logger}
}

func (s *ProductService) CreateProduct(ctx context.Context, product *model.Product) error {
	if product.Name == "" {
		return model.NewValidationError("product name is required")
	}
	if product.Price <= 0 {
		return model.NewValidationError("product price must be greater than zero")
	}
	switch product.Status {
	case model.ProductStatusActive, model.ProductStatusInactive, model.ProductStatusArchived:
	default:
		product.Status = model.ProductStatusActive
	}
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now
	return s.productRepo.Create(ctx, product)
}

func (s *ProductService) GetProduct(ctx context.Context, id, merchantID string) (*model.Product, error) {
	return s.productRepo.GetByID(ctx, id, merchantID)
}

func (s *ProductService) ListProducts(ctx context.Context, merchantID string, page, pageSize int) ([]*model.Product, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.productRepo.ListByMerchant(ctx, merchantID, pageSize, offset)
}

func (s *ProductService) UpdateProduct(ctx context.Context, product *model.Product) error {
	existing, err := s.productRepo.GetByID(ctx, product.ID, product.MerchantID)
	if err != nil {
		return err
	}
	if existing.Status == model.ProductStatusArchived {
		return model.NewValidationError("cannot update an archived product")
	}
	if product.Status != "" && product.Status != existing.Status {
		if existing.Status == model.ProductStatusArchived {
			return model.NewValidationError("cannot transition from archived status")
		}
	}
	if product.Name == "" {
		product.Name = existing.Name
	}
	if product.Price <= 0 {
		product.Price = existing.Price
	}
	if product.Currency == "" {
		product.Currency = existing.Currency
	}
	if product.Description == "" {
		product.Description = existing.Description
	}
	if product.Status == "" {
		product.Status = existing.Status
	}
	if product.Metadata == nil {
		product.Metadata = existing.Metadata
	}
	product.UpdatedAt = time.Now()
	return s.productRepo.Update(ctx, product)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id, merchantID string) error {
	return s.productRepo.Delete(ctx, id, merchantID)
}
