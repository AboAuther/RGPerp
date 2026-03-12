package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/AboAuther/RGPerp/backend/internal/middleware"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
	"github.com/AboAuther/RGPerp/backend/internal/service"
)

type WithdrawalHandler struct {
	withdrawalService *service.WithdrawalService
}

func NewWithdrawalHandler(withdrawalService *service.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{withdrawalService: withdrawalService}
}

type createWithdrawalRequest struct {
	Amount string `json:"amount" binding:"required"`
}

func (h *WithdrawalHandler) Create(c *gin.Context) {
	userIDValue, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		response.FailWithMsg(c, http.StatusUnauthorized, 10004, "unauthorized")
		return
	}
	walletAddressValue, ok := c.Get(middleware.ContextWalletAddressKey)
	if !ok {
		response.FailWithMsg(c, http.StatusUnauthorized, 10004, "unauthorized")
		return
	}

	var req createWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid request payload")
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid amount")
		return
	}

	result, err := h.withdrawalService.Create(service.CreateWithdrawalInput{
		UserID:         userIDValue.(uint64),
		WalletAddress:  walletAddressValue.(string),
		Amount:         amount,
		IdempotencyKey: c.GetHeader("X-Idempotency-Key"),
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, result)
}

func (h *WithdrawalHandler) List(c *gin.Context) {
	userIDValue, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		response.FailWithMsg(c, http.StatusUnauthorized, 10004, "unauthorized")
		return
	}

	items, err := h.withdrawalService.List(userIDValue.(uint64))
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, gin.H{"items": service.BuildWithdrawalList(items)})
}
