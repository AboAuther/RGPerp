package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xiaobao/perpexchange/backend/internal/config"
	"github.com/xiaobao/perpexchange/backend/internal/model"
	"github.com/xiaobao/perpexchange/backend/internal/router"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
		logger, _ = zap.NewProduction()
	}

	db, err := initDB(cfg)
	if err != nil {
		logger.Fatal("failed to connect database", zap.Error(err))
	}
	logger.Info("database connected")

	if err := model.AutoMigrate(db); err != nil {
		logger.Fatal("failed to auto migrate", zap.Error(err))
	}
	if err := model.CreateCompositeIndexes(db); err != nil {
		logger.Warn("failed to create composite indexes (may already exist)", zap.Error(err))
	}
	if err := model.SeedSymbols(db); err != nil {
		logger.Warn("failed to seed symbols", zap.Error(err))
	}
	if err := model.SeedInsuranceFund(db); err != nil {
		logger.Warn("failed to seed insurance fund", zap.Error(err))
	}
	logger.Info("database migration and seed completed")

	rds := initRedis(cfg)
	if err := rds.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("failed to connect redis", zap.Error(err))
	}
	logger.Info("redis connected")

	r := router.Setup(db, rds, logger)

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}
	logger.Info("server exited")
}

func initDB(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	}
	if cfg.Server.Mode == "release" {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Warn)
	}
	return gorm.Open(mysql.Open(cfg.DB.DSN()), gormCfg)
}

func initRedis(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}
