package provider

import (
	"fmt"

	"github.com/helix-seller/helix-seller/internal/config"
)

type Factory struct {
	providers map[string]PaymentProvider
}

func NewFactory(cfg *config.Config) *Factory {
	f := &Factory{providers: make(map[string]PaymentProvider)}
	if cfg.StripeAPIKey != "" {
		f.providers["stripe"] = NewStripeProvider(cfg.StripeAPIKey, cfg.StripeWebhookSecret)
	}
	f.providers["paypal"] = NewPayPalProvider(cfg.PayPalClientID, cfg.PayPalSecret, cfg.PayPalWebhookID)
	f.providers["square"] = NewSquareProvider(cfg.SquareAccessToken, cfg.SquareApplicationID, cfg.SquareWebhookSigKey)
	return f
}

func (f *Factory) Get(name string) (PaymentProvider, error) {
	p, ok := f.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown payment provider: %s", name)
	}
	return p, nil
}
