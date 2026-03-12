package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHealthHandler(db *gorm.DB, rds *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: rds}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	status := gin.H{
		"service": "ok",
		"mysql":   "ok",
		"redis":   "ok",
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		status["mysql"] = err.Error()
	} else if err = sqlDB.PingContext(ctx); err != nil {
		status["mysql"] = err.Error()
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		status["redis"] = err.Error()
	}

	response.OK(c, status)
}
