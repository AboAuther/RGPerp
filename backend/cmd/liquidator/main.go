package main

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/xiaobao/perpexchange/backend/internal/config"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	logger.Info("liquidator process initialized",
		zap.String("price_source", cfg.Price.Source),
		zap.String("db_host", cfg.DB.Host),
		zap.String("redis_addr", cfg.Redis.Addr),
	)

	waitForShutdown(logger, "liquidator")
}

func waitForShutdown(logger *zap.Logger, service string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("service stopped", zap.String("service", service))
}
