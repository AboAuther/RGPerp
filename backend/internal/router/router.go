package router

import (
	"github.com/AboAuther/RGPerp/backend/internal/config"
	"github.com/AboAuther/RGPerp/backend/internal/handler"
	"github.com/AboAuther/RGPerp/backend/internal/middleware"
	"github.com/AboAuther/RGPerp/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, rds *redis.Client, logger *zap.Logger, cfg *config.Config) *gin.Engine {
	r := gin.New()

	r.Use(middleware.CORS())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recovery(logger))

	healthH := handler.NewHealthHandler(db, rds)
	authSvc := service.NewAuthService(db, cfg)
	accountSvc := service.NewAccountService(db)
	withdrawalSvc := service.NewWithdrawalService(db, cfg)
	marketSvc := service.NewMarketService(db)
	orderSvc := service.NewOrderService(db)

	authH := handler.NewAuthHandler(authSvc)
	accountH := handler.NewAccountHandler(accountSvc)
	walletH := handler.NewWalletHandler(cfg)
	withdrawalH := handler.NewWithdrawalHandler(withdrawalSvc)
	marketH := handler.NewMarketHandler(marketSvc)
	orderH := handler.NewOrderHandler(orderSvc)

	r.GET("/health", healthH.Health)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/challenge", authH.Challenge)
	v1.POST("/auth/login", authH.Login)
	v1.GET("/markets", marketH.List)
	v1.GET("/markets/:symbol/price", marketH.Price)
	v1.GET("/markets/:symbol/ticker", marketH.Ticker)
	v1.GET("/markets/:symbol/klines", marketH.Klines)

	authenticated := v1.Group("/")
	authenticated.Use(middleware.Auth(cfg.JWT.Secret))
	authenticated.GET("/account", accountH.Get)
	authenticated.GET("/deposits", accountH.Deposits)
	authenticated.GET("/wallet/deposit-info", walletH.DepositInfo)
	authenticated.POST("/withdrawals", withdrawalH.Create)
	authenticated.GET("/withdrawals", withdrawalH.List)
	authenticated.POST("/orders", orderH.Create)
	authenticated.GET("/orders", orderH.ListOrders)
	authenticated.GET("/positions", orderH.ListPositions)

	return r
}
