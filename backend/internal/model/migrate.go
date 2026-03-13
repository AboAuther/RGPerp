package model

import (
	"fmt"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&AuthNonce{},
		&Account{},
		&LedgerEntry{},
		&Symbol{},
		&Order{},
		&Trade{},
		&Position{},
		&FundingEvent{},
		&VaultEvent{},
		&WithdrawalRequest{},
		&HedgeTask{},
		&HedgeOrder{},
		&Liquidation{},
		&PriceTick{},
		&SystemRiskSnapshot{},
		&InsuranceFund{},
		&SettlementPool{},
	)
}

func CreateCompositeIndexes(db *gorm.DB) error {
	indexes := []struct {
		table   string
		name    string
		columns string
		unique  bool
	}{
		{"vault_events", "uq_vault_events_tx_log", "tx_hash, log_index", true},
		{"orders", "idx_orders_user_status", "user_id, status", false},
		{"orders", "idx_orders_symbol_created", "symbol, created_at", false},
		{"trades", "idx_trades_symbol_created", "symbol, created_at", false},
		{"positions", "idx_positions_user_symbol_status", "user_id, symbol, status", false},
		{"ledger_entries", "idx_ledger_ref", "reference_type, reference_id", false},
		{"funding_events", "uq_funding_position_settlement", "position_id, settlement_at", true},
		{"price_ticks", "idx_price_ticks_symbol_created", "symbol, created_at", false},
		{"system_risk_snapshots", "idx_risk_snapshots_symbol_created", "symbol, created_at", false},
	}

	for _, idx := range indexes {
		if indexExists(db, idx.table, idx.name) {
			continue
		}
		kind := "INDEX"
		if idx.unique {
			kind = "UNIQUE INDEX"
		}
		sql := fmt.Sprintf("CREATE %s %s ON %s (%s)", kind, idx.name, idx.table, idx.columns)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("create index %s: %w", idx.name, err)
		}
	}

	return nil
}

func indexExists(db *gorm.DB, table, indexName string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?",
		table, indexName).Scan(&count)
	return count > 0
}
