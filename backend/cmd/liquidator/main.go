package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/AboAuther/RGPerp/backend/internal/config"
	"github.com/AboAuther/RGPerp/backend/internal/model"
	"github.com/AboAuther/RGPerp/backend/internal/service"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	liquidator := service.NewLiquidatorService(db, logger)
	go liquidator.Run(ctx, 3*time.Second)

	logger.Info("liquidator process initialized",
		zap.String("price_source", cfg.Price.Source),
		zap.String("db_host", cfg.DB.Host),
		zap.String("redis_addr", cfg.Redis.Addr),
	)

	waitForShutdown(logger, "liquidator", cancel)
}

func waitForShutdown(logger *zap.Logger, service string, cancel context.CancelFunc) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	logger.Info("service stopped", zap.String("service", service))
}
