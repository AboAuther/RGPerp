package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/AboAuther/RGPerp/backend/internal/config"
	backendindexer "github.com/AboAuther/RGPerp/backend/internal/indexer"
	"github.com/AboAuther/RGPerp/backend/internal/model"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	db, err := gorm.Open(mysql.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		logger.Fatal("failed to connect database", zap.Error(err))
	}
	if err := model.AutoMigrate(db); err != nil {
		logger.Fatal("failed to migrate database", zap.Error(err))
	}

	client, err := ethclient.Dial(cfg.Blockchain.RPCURL)
	if err != nil {
		logger.Fatal("failed to connect rpc", zap.Error(err))
	}

	listener, err := backendindexer.NewListener(db, client, cfg.Blockchain.VaultAddress, logger)
	if err != nil {
		logger.Fatal("failed to initialize listener", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := listener.Run(ctx); err != nil {
			logger.Fatal("indexer stopped unexpectedly", zap.Error(err))
		}
	}()

	logger.Info("indexer process initialized",
		zap.String("rpc_url", cfg.Blockchain.RPCURL),
		zap.String("vault_address", cfg.Blockchain.VaultAddress),
		zap.Int64("chain_id", cfg.Blockchain.ChainID),
	)

	waitForShutdown(logger, "indexer", cancel)
}

func waitForShutdown(logger *zap.Logger, service string, cancel context.CancelFunc) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	logger.Info("service stopped", zap.String("service", service))
}
