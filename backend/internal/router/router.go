package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/xiaobao/perpexchange/backend/internal/handler"
	"github.com/xiaobao/perpexchange/backend/internal/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, rds *redis.Client, logger *zap.Logger) *gin.Engine {
	r := gin.New()

	r.Use(middleware.CORS())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recovery(logger))

	healthH := handler.NewHealthHandler(db, rds)
	r.GET("/health", healthH.Health)

	v1 := r.Group("/api/v1")
	_ = v1

	return r
}
