package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AboAuther/RGPerp/backend/internal/middleware"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
	"github.com/AboAuther/RGPerp/backend/internal/service"
)

type AccountHandler struct {
	accountService *service.AccountService
}

func NewAccountHandler(accountService *service.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

func (h *AccountHandler) Get(c *gin.Context) {
	userIDValue, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		response.FailWithMsg(c, http.StatusUnauthorized, 10004, "unauthorized")
		return
	}

	account, err := h.accountService.GetAccount(userIDValue.(uint64))
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, account)
}
