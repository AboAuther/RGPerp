package middleware

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
	"github.com/AboAuther/RGPerp/backend/internal/service"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		wallet, _ := c.Get(ContextWalletAddressKey)
		if !service.IsAdminWallet(toString(wallet)) {
			response.Fail(c, apperr.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Next()
	}
}

func toString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}
