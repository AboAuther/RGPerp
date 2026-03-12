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

	logger.Info("indexer process initialized",
		zap.String("rpc_url", cfg.Blockchain.RPCURL),
		zap.String("vault_address", cfg.Blockchain.VaultAddress),
		zap.Int64("chain_id", cfg.Blockchain.ChainID),
	)

	waitForShutdown(logger, "indexer")
}

func waitForShutdown(logger *zap.Logger, service string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("service stopped", zap.String("service", service))
}
