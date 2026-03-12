package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/AboAuther/RGPerp/backend/internal/config"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
)

type WalletHandler struct {
	cfg *config.Config
}

func NewWalletHandler(cfg *config.Config) *WalletHandler {
	return &WalletHandler{cfg: cfg}
}

func (h *WalletHandler) DepositInfo(c *gin.Context) {
	response.OK(c, gin.H{
		"chain_id":      h.cfg.Blockchain.ChainID,
		"vault_address": h.cfg.Blockchain.VaultAddress,
		"usdc_address":  h.cfg.Blockchain.USDCAddress,
		"asset":         "USDC",
		"decimals":      6,
	})
}
