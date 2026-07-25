package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helix-seller/helix-seller/internal/model"
	"github.com/helix-seller/helix-seller/internal/service"
)

type WebhookDeliveryHandler struct {
	deliverySvc *service.WebhookDeliveryService
}

func NewWebhookDeliveryHandler(deliverySvc *service.WebhookDeliveryService) *WebhookDeliveryHandler {
	return &WebhookDeliveryHandler{deliverySvc: deliverySvc}
}

func (h *WebhookDeliveryHandler) ListDeliveries(c *gin.Context) {
	merchantID, err := uuid.Parse(c.Param("merchantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	deliveries, total, err := h.deliverySvc.ListDeliveries(c.Request.Context(), merchantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list deliveries"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deliveries": deliveries, "total": total})
}

func (h *WebhookDeliveryHandler) GetDelivery(c *gin.Context) {
	id, err := uuid.Parse(c.Param("deliveryId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid delivery id"})
		return
	}

	delivery, err := h.deliverySvc.GetDelivery(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "delivery not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get delivery"})
		return
	}
	c.JSON(http.StatusOK, delivery)
}
