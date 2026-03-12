package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
	"github.com/AboAuther/RGPerp/backend/internal/service"
)

type MarketHandler struct {
	marketService *service.MarketService
}

func NewMarketHandler(marketService *service.MarketService) *MarketHandler {
	return &MarketHandler{marketService: marketService}
}

func (h *MarketHandler) List(c *gin.Context) {
	markets, err := h.marketService.ListMarkets()
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, markets)
}

func (h *MarketHandler) Ticker(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "symbol is required")
		return
	}

	ticker, err := h.marketService.GetTicker(symbol)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ticker)
}
