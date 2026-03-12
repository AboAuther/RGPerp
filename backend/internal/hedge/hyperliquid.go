package hedge

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

func (a *HyperliquidAdapter) PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResult, error) {
	resp, err := a.runBridge(ctx, map[string]any{
		"action":         "order",
		"api_url":        a.apiURL,
		"wallet_address": a.walletAddress,
		"private_key":    a.privateKey,
		"symbol":         strings.ToUpper(strings.TrimSpace(req.Symbol)),
		"side":           strings.ToLower(strings.TrimSpace(req.Side)),
		"size":           req.Size.String(),
		"price":          req.Price.String(),
		"leverage":       5,
		"is_cross":       true,
	})
	if err != nil {
		return nil, err
	}

	status, _ := resp["status"].(string)
	if status == "error" {
		return nil, fmt.Errorf("%v", resp["error"])
	}

	filledSize, _ := decimal.NewFromString(fmt.Sprintf("%v", resp["filled_size"]))
	filledPrice, _ := decimal.NewFromString(fmt.Sprintf("%v", resp["filled_price"]))
	return &OrderResult{
		ExternalOrderID: fmt.Sprintf("%v", resp["external_order_id"]),
		Status:          fmt.Sprintf("%v", resp["order_status"]),
		FilledSize:      filledSize,
		FilledPrice:     filledPrice,
	}, nil
}

func (a *HyperliquidAdapter) GetPosition(ctx context.Context, symbol string) (decimal.Decimal, error) {
	resp, err := a.runBridge(ctx, map[string]any{
		"action":         "position",
		"api_url":        a.apiURL,
		"wallet_address": a.walletAddress,
		"private_key":    a.privateKey,
		"symbol":         strings.ToUpper(strings.TrimSpace(symbol)),
	})
	if err != nil {
		return decimal.Zero, err
	}

	status, _ := resp["status"].(string)
	if status == "error" {
		return decimal.Zero, fmt.Errorf("%v", resp["error"])
	}
	return decimal.NewFromString(fmt.Sprintf("%v", resp["position"]))
}
