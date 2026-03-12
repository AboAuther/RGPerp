package hedge

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/AboAuther/RGPerp/backend/internal/config"
)

type OrderRequest struct {
	Symbol string
	Side   string
	Size   decimal.Decimal
	Price  decimal.Decimal
}

type OrderResult struct {
	ExternalOrderID string
	Status          string
	FilledSize      decimal.Decimal
	FilledPrice     decimal.Decimal
}

type Adapter interface {
	PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResult, error)
}

type MockAdapter struct{}

type HyperliquidAdapter struct {
	apiURL        string
	walletAddress string
	privateKey    string
	client        *http.Client
}

func NewAdapter(cfg *config.Config) Adapter {
	if useHyperliquidAdapter(cfg) {
		return &HyperliquidAdapter{
			apiURL:        strings.TrimRight(cfg.Hyperliquid.APIURL, "/"),
			walletAddress: cfg.Hyperliquid.WalletAddress,
			privateKey:    cfg.Hyperliquid.PrivateKey,
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}
	return &MockAdapter{}
}

func (a *MockAdapter) PlaceOrder(_ context.Context, req OrderRequest) (*OrderResult, error) {
	return &OrderResult{
		ExternalOrderID: fmt.Sprintf("mock-%s-%s", strings.ToLower(req.Symbol), req.Side),
		Status:          "filled",
		FilledSize:      req.Size,
		FilledPrice:     req.Price,
	}, nil
}

func useHyperliquidAdapter(cfg *config.Config) bool {
	apiURL := strings.TrimSpace(cfg.Hyperliquid.APIURL)
	wallet := strings.TrimSpace(strings.ToLower(cfg.Hyperliquid.WalletAddress))
	privateKey := strings.TrimSpace(strings.ToLower(cfg.Hyperliquid.PrivateKey))
	if apiURL == "" {
		return false
	}
	if wallet == "" || wallet == "0x0000000000000000000000000000000000000000" {
		return false
	}
	if privateKey == "" || privateKey == "0x0000000000000000000000000000000000000000000000000000000000000000" {
		return false
	}
	return true
}
