package indexer

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Listener struct {
	db            *gorm.DB
	client        *ethclient.Client
	vaultAddress  common.Address
	processor     *Processor
	logger        *zap.Logger
	pollInterval  time.Duration
	lastProcessed uint64
}

func NewListener(db *gorm.DB, client *ethclient.Client, vaultAddress string, logger *zap.Logger) (*Listener, error) {
	processor, err := NewProcessor(db)
	if err != nil {
		return nil, err
	}

	l := &Listener{
		db:           db,
		client:       client,
		vaultAddress: common.HexToAddress(vaultAddress),
		processor:    processor,
		logger:       logger,
		pollInterval: 5 * time.Second,
	}

	if err := l.loadLastProcessed(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Listener) Run(ctx context.Context) error {
	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()

	if err := l.sync(ctx); err != nil {
		l.logger.Warn("initial sync failed", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := l.sync(ctx); err != nil {
				l.logger.Warn("sync failed", zap.Error(err))
			}
		}
	}
}

func (l *Listener) loadLastProcessed() error {
	type blockResult struct {
		BlockNumber uint64
	}
	var result blockResult
	err := l.db.Table("vault_events").Select("COALESCE(MAX(block_number), 0) AS block_number").Scan(&result).Error
	if err != nil {
		return err
	}
	l.lastProcessed = result.BlockNumber
	return nil
}

func (l *Listener) sync(ctx context.Context) error {
	header, err := l.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("latest header: %w", err)
	}
	latest := header.Number.Uint64()
	if latest <= l.lastProcessed {
		return nil
	}

	fromBlock := l.lastProcessed + 1
	query := ethereum.FilterQuery{
		FromBlock: newBlockNumber(fromBlock),
		ToBlock:   newBlockNumber(latest),
		Addresses: []common.Address{l.vaultAddress},
	}

	logs, err := l.client.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("filter logs: %w", err)
	}

	var firstFailedBlock uint64
	for _, vLog := range logs {
		if len(vLog.Topics) == 0 {
			continue
		}
		if err := l.processor.ProcessLog(ctx, vLog); err != nil {
			if firstFailedBlock == 0 || vLog.BlockNumber < firstFailedBlock {
				firstFailedBlock = vLog.BlockNumber
			}
			l.logger.Warn("process vault log failed",
				zap.Error(err),
				zap.String("tx_hash", vLog.TxHash.Hex()),
				zap.Uint64("block_number", vLog.BlockNumber),
			)
			continue
		}
	}

	if len(logs) == 0 {
		l.lastProcessed = latest
		return nil
	}

	if firstFailedBlock > 0 {
		// Rewind to the previous block so failed logs are retried in the next poll.
		// Dedup on (tx_hash, log_index) keeps already-processed events idempotent.
		if firstFailedBlock > 0 {
			l.lastProcessed = firstFailedBlock - 1
		}
		return fmt.Errorf("vault log processing failed at block %d", firstFailedBlock)
	}

	l.lastProcessed = latest
	return nil
}

func newBlockNumber(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}
