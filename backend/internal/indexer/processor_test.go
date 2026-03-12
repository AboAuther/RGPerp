package indexer

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

func TestProcessDeposit_IdempotentOnDuplicateLog(t *testing.T) {
	db := mustNewTestDB(t)
	processor, err := NewProcessor(db)
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}

	user := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	logItem := buildDepositLog(t, processor, user, 10_000_000, 1, 0)

	ctx := context.Background()
	if err := processor.ProcessLog(ctx, logItem); err != nil {
		t.Fatalf("first process failed: %v", err)
	}
	if err := processor.ProcessLog(ctx, logItem); err != nil {
		t.Fatalf("second process failed: %v", err)
	}

	var account model.Account
	if err := db.Where("user_id = ?", uint64(1)).First(&account).Error; err != nil {
		t.Fatalf("load account: %v", err)
	}
	if !account.AvailableBalance.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("unexpected balance: got %s", account.AvailableBalance.String())
	}

	var events int64
	if err := db.Model(&model.VaultEvent{}).Count(&events).Error; err != nil {
		t.Fatalf("count vault events: %v", err)
	}
	if events != 1 {
		t.Fatalf("unexpected vault event count: %d", events)
	}

	var ledgers int64
	if err := db.Model(&model.LedgerEntry{}).Where("type = ?", "deposit").Count(&ledgers).Error; err != nil {
		t.Fatalf("count ledgers: %v", err)
	}
	if ledgers != 1 {
		t.Fatalf("unexpected ledger count: %d", ledgers)
	}
}

func TestProcessDeposit_DifferentLogsAccumulateBalance(t *testing.T) {
	db := mustNewTestDB(t)
	processor, err := NewProcessor(db)
	if err != nil {
		t.Fatalf("new processor: %v", err)
	}

	user := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	log1 := buildDepositLog(t, processor, user, 10_000_000, 2, 0)
	log2 := buildDepositLog(t, processor, user, 12_500_000, 3, 1)

	ctx := context.Background()
	if err := processor.ProcessLog(ctx, log1); err != nil {
		t.Fatalf("process log1 failed: %v", err)
	}
	if err := processor.ProcessLog(ctx, log2); err != nil {
		t.Fatalf("process log2 failed: %v", err)
	}

	var account model.Account
	if err := db.Where("user_id = ?", uint64(1)).First(&account).Error; err != nil {
		t.Fatalf("load account: %v", err)
	}
	if !account.AvailableBalance.Equal(decimal.RequireFromString("22.5")) {
		t.Fatalf("unexpected balance: got %s", account.AvailableBalance.String())
	}
}

func mustNewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uq_vault_events_tx_log ON vault_events (tx_hash, log_index)").Error; err != nil {
		t.Fatalf("create unique index: %v", err)
	}
	return db
}

func buildDepositLog(t *testing.T, processor *Processor, user common.Address, amount uint64, block uint64, idx uint) gethtypes.Log {
	t.Helper()

	data, err := processor.parsedABI.Events["Deposit"].Inputs.NonIndexed().Pack(
		new(big.Int).SetUint64(amount),
		new(big.Int).SetUint64(uint64(time.Now().Unix())),
	)
	if err != nil {
		t.Fatalf("pack deposit event: %v", err)
	}

	return gethtypes.Log{
		Address:     common.HexToAddress("0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512"),
		Topics:      []common.Hash{processor.depositTopic, common.BytesToHash(common.LeftPadBytes(user.Bytes(), 32))},
		Data:        data,
		BlockNumber: block,
		TxHash:      common.BigToHash(new(big.Int).SetUint64(block*100 + uint64(idx))),
		Index:       idx,
	}
}
