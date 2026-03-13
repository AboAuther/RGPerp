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
	if err := model.CreateCompositeIndexes(db); err != nil {
		logger.Warn("failed to create composite indexes", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := service.NewFundingService(db, logger)
	go svc.Run(ctx, 15*time.Second)

	logger.Info("funding settlement process initialized",
		zap.String("price_source", cfg.Price.Source),
		zap.Duration("interval", 15*time.Second),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	logger.Info("service stopped", zap.String("service", "fundingd"))
}
