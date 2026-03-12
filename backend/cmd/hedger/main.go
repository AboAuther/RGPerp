package main

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/AboAuther/RGPerp/backend/internal/config"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	logger.Info("hedger process initialized",
		zap.String("hyperliquid_api_url", cfg.Hyperliquid.APIURL),
		zap.String("wallet_address", cfg.Hyperliquid.WalletAddress),
		zap.String("price_source", cfg.Price.Source),
	)

	waitForShutdown(logger, "hedger")
}

func waitForShutdown(logger *zap.Logger, service string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("service stopped", zap.String("service", service))
}
