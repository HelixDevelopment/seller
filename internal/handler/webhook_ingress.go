package handler

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/helix-seller/helix-seller/internal/eventbus"
	"github.com/helix-seller/helix-seller/internal/service"
)

type WebhookIngressHandler struct {
	webhookSvc         *service.WebhookService
	eventBus           eventbus.EventBus
	logger             *zap.Logger
	stripeWebhookSecret string
	paypalWebhookID     string
	squareWebhookSigKey string
}

func NewWebhookIngressHandler(webhookSvc *service.WebhookService, eventBus eventbus.EventBus, logger *zap.Logger, stripeWebhookSecret, paypalWebhookID, squareWebhookSigKey string) *WebhookIngressHandler {
	return &WebhookIngressHandler{
		webhookSvc:          webhookSvc,
		eventBus:            eventBus,
		logger:              logger,
		stripeWebhookSecret: stripeWebhookSecret,
		paypalWebhookID:     paypalWebhookID,
		squareWebhookSigKey: squareWebhookSigKey,
	}
}

func verifyStripeSignature(payload []byte, sigHeader string, secret string) error {
	if secret == "" {
		return fmt.Errorf("stripe webhook secret not configured")
	}
	if sigHeader == "" {
		return fmt.Errorf("missing Stripe-Signature header")
	}

	expectedSig := ""
	timestamp := ""

	for _, part := range strings.Split(sigHeader, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "v1=") {
			expectedSig = part[3:]
		} else if strings.HasPrefix(part, "t=") {
			timestamp = part[2:]
		}
	}

	if expectedSig == "" || timestamp == "" {
		return fmt.Errorf("invalid stripe signature header format")
	}

	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	computedSig := hex.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(computedSig), []byte(expectedSig)) != 1 {
		return fmt.Errorf("stripe signature mismatch")
	}

	return nil
}

func verifySquareSignature(payload []byte, sigHeader string, sigKey string) error {
	if sigKey == "" {
		return fmt.Errorf("square webhook signature key not configured")
	}
	if sigHeader == "" {
		return fmt.Errorf("missing x-square-hmacsha256-signature header")
	}

	mac := hmac.New(sha256.New, []byte(sigKey))
	mac.Write(payload)
	computedSig := hex.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(computedSig), []byte(sigHeader)) != 1 {
		return fmt.Errorf("square signature mismatch")
	}

	return nil
}

func verifyPayPalWebhookSignature(payload []byte, headers map[string]string) error {
	certURL := headers["PAYPAL-CERT-URL"]
	transmissionSig := headers["PAYPAL-TRANSMISSION-SIG"]
	transmissionID := headers["PAYPAL-TRANSMISSION-ID"]
	transmissionTime := headers["PAYPAL-TRANSMISSION-TIME"]

	if certURL == "" || transmissionSig == "" || transmissionID == "" || transmissionTime == "" {
		return fmt.Errorf("missing required PayPal webhook headers")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(certURL)
	if err != nil {
		return fmt.Errorf("fetch PayPal certificate: %w", err)
	}
	defer resp.Body.Close()

	certPEM, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read PayPal certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PayPal certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse PayPal certificate: %w", err)
	}

	sigString := transmissionID + "|" + transmissionTime + "|" + string(payload)

	sig, err := base64.StdEncoding.DecodeString(transmissionSig)
	if err != nil {
		return fmt.Errorf("decode PayPal signature: %w", err)
	}

	rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("PayPal certificate public key is not RSA")
	}

	hash := crypto.SHA256
	h := hash.New()
	h.Write([]byte(sigString))
	if err := rsa.VerifyPKCS1v15(rsaPub, hash, h.Sum(nil), sig); err != nil {
		return fmt.Errorf("PayPal signature verification failed: %w", err)
	}

	return nil
}

func (h *WebhookIngressHandler) HandleStripe(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	sig := c.GetHeader("Stripe-Signature")
	if err := verifyStripeSignature(body, sig, h.stripeWebhookSecret); err != nil {
		h.logger.Warn("stripe webhook signature verification failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	h.eventBus.Publish(c.Request.Context(), "events.provider.stripe", &eventbus.Event{
		Type:   "provider.webhook.received",
		Source: "stripe",
		Data:   string(body),
	})
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *WebhookIngressHandler) HandlePayPal(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// PayPal webhook verification requires:
	// 1. Fetching the certificate from PAYPAL-CERT-URL
	// 2. Verifying PAYPAL-TRANSMISSION-SIG against the certificate
	// 3. Checking transmission time is within 5 minutes
	// 4. Confirming the webhook ID matches
	// Full implementation requires PayPal SDK. Basic header validation below.
	transmissionSig := c.GetHeader("PAYPAL-TRANSMISSION-SIG")
	transmissionID := c.GetHeader("PAYPAL-TRANSMISSION-ID")
	transmissionTime := c.GetHeader("PAYPAL-TRANSMISSION-TIME")

	if h.paypalWebhookID == "" {
		h.logger.Warn("paypal webhook id not configured, rejecting webhook")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook not configured"})
		return
	}

	if transmissionSig == "" || transmissionID == "" || transmissionTime == "" {
		h.logger.Warn("missing paypal webhook headers")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing webhook headers"})
		return
	}

	// Verify transmission time is within 5 minutes
	parsedTime, err := time.Parse(time.RFC3339, transmissionTime)
	if err != nil {
		h.logger.Warn("invalid paypal transmission time", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid transmission time"})
		return
	}
	if time.Since(parsedTime).Abs() > 5*time.Minute {
		h.logger.Warn("paypal transmission time outside tolerance")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "transmission time outside tolerance"})
		return
	}

	headers := map[string]string{
		"PAYPAL-CERT-URL":        c.GetHeader("PAYPAL-CERT-URL"),
		"PAYPAL-TRANSMISSION-SIG": c.GetHeader("PAYPAL-TRANSMISSION-SIG"),
		"PAYPAL-TRANSMISSION-ID":  c.GetHeader("PAYPAL-TRANSMISSION-ID"),
		"PAYPAL-TRANSMISSION-TIME": c.GetHeader("PAYPAL-TRANSMISSION-TIME"),
		"PAYPAL-AUTH-ALGO":        c.GetHeader("PAYPAL-AUTH-ALGO"),
	}

	if err := verifyPayPalWebhookSignature(body, headers); err != nil {
		h.logger.Warn("paypal webhook signature verification failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.eventBus.Publish(c.Request.Context(), "events.provider.paypal", &eventbus.Event{
		Type:   "provider.webhook.received",
		Source: "paypal",
		Data:   string(body),
	})
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *WebhookIngressHandler) HandleSquare(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	sig := c.GetHeader("x-square-hmacsha256-signature")
	if err := verifySquareSignature(body, sig, h.squareWebhookSigKey); err != nil {
		h.logger.Warn("square webhook signature verification failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	h.eventBus.Publish(c.Request.Context(), "events.provider.square", &eventbus.Event{
		Type:   "provider.webhook.received",
		Source: "square",
		Data:   string(body),
	})
	c.JSON(http.StatusOK, gin.H{"received": true})
}
