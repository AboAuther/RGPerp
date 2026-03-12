package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/AboAuther/RGPerp/backend/internal/pkg/authjwt"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
)

const (
	ContextUserIDKey        = "user_id"
	ContextWalletAddressKey = "wallet_address"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Fail(c, apperr.ErrUnauthorized)
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := authjwt.Parse(secret, token)
		if err != nil {
			response.Fail(c, apperr.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextWalletAddressKey, claims.WalletAddress)
		c.Next()
	}
}
