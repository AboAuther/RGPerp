package hedge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
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
	GetPosition(ctx context.Context, symbol string) (decimal.Decimal, error)
}

type MockAdapter struct{}

type HyperliquidAdapter struct {
	apiURL        string
	walletAddress string
	privateKey    string
	client        *http.Client
	bridgePath    string
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
			bridgePath: "scripts/hyperliquid_bridge.py",
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

func (a *MockAdapter) GetPosition(_ context.Context, _ string) (decimal.Decimal, error) {
	return decimal.Zero, nil
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

func (a *HyperliquidAdapter) runBridge(ctx context.Context, payload map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "python3", a.bridgePath)
	cmd.Stdin = strings.NewReader(string(raw))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("hyperliquid bridge failed: %s", strings.TrimSpace(string(out)))
	}

	var data map[string]any
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}
	return data, nil
}
