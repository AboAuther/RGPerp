package hedge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
)

type hyperliquidOrderAction struct {
	Type     string                    `json:"type"`
	Orders   []hyperliquidOrderRequest `json:"orders"`
	Grouping string                    `json:"grouping"`
}

type hyperliquidOrderRequest struct {
	Asset      int                    `json:"a"`
	IsBuy      bool                   `json:"b"`
	Price      string                 `json:"p"`
	Size       string                 `json:"s"`
	ReduceOnly bool                   `json:"r"`
	OrderType  map[string]interface{} `json:"t"`
}

type hyperliquidExchangeRequest struct {
	Action    hyperliquidOrderAction `json:"action"`
	Nonce     int64                  `json:"nonce"`
	Signature map[string]string      `json:"signature"`
	VaultAddr *string                `json:"vaultAddress"`
}

func (a *HyperliquidAdapter) PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResult, error) {
	if a.client == nil {
		return nil, apperr.ErrInternal
	}
	payload, err := a.buildExchangePayload(req)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL+"/exchange", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hyperliquid exchange returned status %d", resp.StatusCode)
	}

	return &OrderResult{
		ExternalOrderID: fmt.Sprintf("hl-%s-%s", strings.ToLower(req.Symbol), req.Side),
		Status:          "submitted",
		FilledSize:      req.Size,
		FilledPrice:     req.Price,
	}, nil
}

func (a *HyperliquidAdapter) buildExchangePayload(req OrderRequest) (*hyperliquidExchangeRequest, error) {
	return nil, fmt.Errorf("hyperliquid signing flow not connected yet")
}
