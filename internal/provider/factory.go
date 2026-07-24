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
	if cfg.ProviderMock || !hasRealKeys(cfg) {
		mock := NewMockProvider()
		f.providers["mock"] = mock
		f.providers["stripe"] = mock
		f.providers["paypal"] = mock
		f.providers["square"] = mock
		return f
	}
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

func hasRealKeys(cfg *config.Config) bool {
	return cfg.StripeAPIKey != "" || cfg.PayPalClientID != "" || cfg.SquareAccessToken != ""
}
