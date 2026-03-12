package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/AboAuther/RGPerp/backend/internal/middleware"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
	"github.com/AboAuther/RGPerp/backend/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

type createOrderRequest struct {
	ClientOrderID string `json:"client_order_id"`
	Symbol        string `json:"symbol" binding:"required"`
	Side          string `json:"side" binding:"required"`
	Type          string `json:"type" binding:"required"`
	Size          string `json:"size" binding:"required"`
	Leverage      uint32 `json:"leverage" binding:"required"`
	Margin        string `json:"margin"`
	ReduceOnly    bool   `json:"reduce_only"`
}

func (h *OrderHandler) Create(c *gin.Context) {
	userIDValue, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		response.FailWithMsg(c, http.StatusUnauthorized, 10004, "unauthorized")
		return
	}

	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid request payload")
		return
	}

	size, err := decimal.NewFromString(req.Size)
	if err != nil {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid size")
		return
	}

	margin := decimal.Zero
	if req.Margin != "" {
		margin, err = decimal.NewFromString(req.Margin)
		if err != nil {
			response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid margin")
			return
		}
	}

	result, err := h.orderService.Create(service.CreateOrderInput{
		UserID:        userIDValue.(uint64),
		ClientOrderID: req.ClientOrderID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Type:          req.Type,
		Size:          size,
		Leverage:      req.Leverage,
		Margin:        margin,
		ReduceOnly:    req.ReduceOnly,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *OrderHandler) ListPositions(c *gin.Context) {
	userIDValue, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		response.FailWithMsg(c, http.StatusUnauthorized, 10004, "unauthorized")
		return
	}

	items, err := h.orderService.ListOpenPositions(userIDValue.(uint64))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	userIDValue, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		response.FailWithMsg(c, http.StatusUnauthorized, 10004, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.orderService.ListOrders(userIDValue.(uint64), limit)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}
