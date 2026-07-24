package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helix-seller/helix-seller/internal/model"
	"github.com/helix-seller/helix-seller/internal/service"
)

type ProductHandler struct {
	productSvc *service.ProductService
}

func NewProductHandler(productSvc *service.ProductService) *ProductHandler {
	return &ProductHandler{productSvc: productSvc}
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	merchantID := c.Param("merchantId")
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}
	var req struct {
		Name        string          `json:"name" binding:"required"`
		Description string          `json:"description"`
		Price       int64           `json:"price" binding:"required,gt=0"`
		Currency    string          `json:"currency" binding:"required"`
		Status      string          `json:"status"`
		Metadata    json.RawMessage `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product := &model.Product{
		ID:          uuid.New().String(),
		MerchantID:  merchantID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Currency:    req.Currency,
		Status:      model.ProductStatus(req.Status),
		Metadata:    json.RawMessage(`{}`),
	}
	if err := h.productSvc.CreateProduct(c.Request.Context(), product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}
	c.JSON(http.StatusCreated, product)
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	merchantID := c.Param("merchantId")
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}
	page, pageSize := 1, 20
	products, total, err := h.productSvc.ListProducts(c.Request.Context(), merchantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products, "total": total})
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	merchantID := c.Param("merchantId")
	productID := c.Param("productId")
	if merchantID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	product, err := h.productSvc.GetProduct(c.Request.Context(), productID, merchantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	merchantID := c.Param("merchantId")
	productID := c.Param("productId")
	if merchantID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Price       *int64          `json:"price"`
		Currency    string          `json:"currency"`
		Status      string          `json:"status"`
		Metadata    json.RawMessage `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing, err := h.productSvc.GetProduct(c.Request.Context(), productID, merchantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Price != nil {
		existing.Price = *req.Price
	}
	if req.Currency != "" {
		existing.Currency = req.Currency
	}
	if req.Status != "" {
		existing.Status = model.ProductStatus(req.Status)
	}
	if req.Metadata != nil {
		existing.Metadata = req.Metadata
	}
	if err := h.productSvc.UpdateProduct(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	merchantID := c.Param("merchantId")
	productID := c.Param("productId")
	if merchantID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.productSvc.DeleteProduct(c.Request.Context(), productID, merchantID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "product deleted"})
}
