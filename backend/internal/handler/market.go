package handler

import (
	"net/http"
	"strconv"
	"time"

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

func (h *MarketHandler) Price(c *gin.Context) {
	h.Ticker(c)
}

func (h *MarketHandler) Klines(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1m")
	limit := parseIntDefault(c.Query("limit"), 200)

	start, err := parseUnixTime(c.Query("start_time"))
	if err != nil {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid start_time")
		return
	}
	end, err := parseUnixTime(c.Query("end_time"))
	if err != nil {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid end_time")
		return
	}

	items, err := h.marketService.GetKlines(symbol, interval, start, end, limit)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

func parseIntDefault(v string, d int) int {
	if v == "" {
		return d
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return n
}

func parseUnixTime(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	// Accept both milliseconds and seconds.
	if n > 1_000_000_000_000 {
		return time.UnixMilli(n).UTC(), nil
	}
	return time.Unix(n, 0).UTC(), nil
}
