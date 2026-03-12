# 数据库 Schema 设计 (GORM)

本文档定义所有 GORM 模型。数据库使用 MySQL 8.0，所有金额字段使用 `DECIMAL(36,18)` 存储以保持高精度，Go 代码中使用 `shopspring/decimal`。

## 公共基础模型

```go
type BaseModel struct {
    ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
    CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}
```

---

## 1. users — 用户表

```go
type User struct {
    BaseModel
    WalletAddress string `gorm:"type:varchar(42);uniqueIndex;not null" json:"wallet_address"`
    Status        string `gorm:"type:varchar(20);default:'active';not null" json:"status"`
    // Status 枚举: active / frozen / banned
}
```

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | |
| wallet_address | VARCHAR(42) | UNIQUE, NOT NULL | EVM 钱包地址 (含 0x) |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'active' | active / frozen / banned |
| created_at | DATETIME(3) | | |
| updated_at | DATETIME(3) | | |

---

## 2. auth_nonces — 登录挑战表

```go
type AuthNonce struct {
    BaseModel
    WalletAddress string    `gorm:"type:varchar(42);index;not null" json:"wallet_address"`
    Nonce         string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"nonce"`
    Message       string    `gorm:"type:text;not null" json:"message"`
    Domain        string    `gorm:"type:varchar(255);not null" json:"domain"`
    ChainID       uint64    `gorm:"not null" json:"chain_id"`
    ExpiresAt     time.Time `gorm:"index;not null" json:"expires_at"`
    Used          bool      `gorm:"default:false;not null" json:"used"`
}
```

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | BIGINT UNSIGNED | PK | |
| wallet_address | VARCHAR(42) | INDEX, NOT NULL | |
| nonce | VARCHAR(64) | UNIQUE, NOT NULL | 随机挑战值 |
| message | TEXT | NOT NULL | 完整待签名消息 |
| domain | VARCHAR(255) | NOT NULL | 签名域名 |
| chain_id | BIGINT UNSIGNED | NOT NULL | 链 ID |
| expires_at | DATETIME(3) | INDEX, NOT NULL | 过期时间 |
| used | TINYINT(1) | NOT NULL, DEFAULT 0 | 是否已消费 |

---

## 3. accounts — 用户账户表

```go
type Account struct {
    BaseModel
    UserID           uint64          `gorm:"uniqueIndex;not null" json:"user_id"`
    AvailableBalance decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"available_balance"`
    LockedBalance    decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"locked_balance"`
    Version          uint64          `gorm:"default:0;not null" json:"version"`

    User User `gorm:"foreignKey:UserID" json:"-"`
}
```

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | BIGINT UNSIGNED | PK | |
| user_id | BIGINT UNSIGNED | UNIQUE, NOT NULL, FK(users) | |
| available_balance | DECIMAL(36,18) | NOT NULL, DEFAULT 0 | 可用余额 |
| locked_balance | DECIMAL(36,18) | NOT NULL, DEFAULT 0 | 保证金占用 |
| version | BIGINT UNSIGNED | NOT NULL, DEFAULT 0 | 乐观锁版本号 |

---

## 4. ledger_entries — 账本流水表

```go
type LedgerEntry struct {
    BaseModel
    UserID        uint64          `gorm:"index;not null" json:"user_id"`
    Type          string          `gorm:"type:varchar(30);index;not null" json:"type"`
    Amount        decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
    BalanceBefore decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"balance_before"`
    BalanceAfter  decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"balance_after"`
    ReferenceType string          `gorm:"type:varchar(30)" json:"reference_type"`
    ReferenceID   uint64          `gorm:"index" json:"reference_id"`
    Description   string          `gorm:"type:varchar(255)" json:"description"`
}
```

Type 枚举：

| 值 | 说明 |
| --- | --- |
| `deposit` | 充值入账 |
| `withdraw` | 提现扣款 |
| `margin_lock` | 开仓保证金冻结 |
| `margin_release` | 平仓保证金释放 |
| `realized_pnl` | 已实现盈亏结算 |
| `liquidation` | 清算扣款 |
| `fee` | 手续费 |
| `funding` | 资金费率 |
| `insurance` | 保险基金 |

---

## 5. symbols — 交易标的配置表

```go
type Symbol struct {
    BaseModel
    Name                  string          `gorm:"type:varchar(20);uniqueIndex;not null" json:"name"`
    BaseAsset             string          `gorm:"type:varchar(10);not null" json:"base_asset"`
    QuoteAsset            string          `gorm:"type:varchar(10);not null" json:"quote_asset"`
    Status                string          `gorm:"type:varchar(20);default:'trading';not null" json:"status"`
    MaxLeverage           uint32          `gorm:"not null" json:"max_leverage"`
    MinOrderSize          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"min_order_size"`
    MaxPositionNotional   decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"max_position_notional"`
    TickSize              decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"tick_size"`
    LotSize               decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"lot_size"`
    InitialMarginRate     decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"initial_margin_rate"`
    MaintenanceMarginRate decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"maintenance_margin_rate"`
    MakerFeeRate          decimal.Decimal `gorm:"type:decimal(10,6);default:0;not null" json:"maker_fee_rate"`
    TakerFeeRate          decimal.Decimal `gorm:"type:decimal(10,6);default:0;not null" json:"taker_fee_rate"`
    HyperliquidAssetIndex int             `gorm:"not null" json:"hyperliquid_asset_index"`
}
```

Status 枚举: `trading` / `paused` / `delisted`

初始数据：

```go
Symbol{
    Name:                  "BTC-PERP",
    BaseAsset:             "BTC",
    QuoteAsset:            "USDC",
    MaxLeverage:           50,
    MinOrderSize:          decimal.NewFromFloat(0.001),
    MaxPositionNotional:   decimal.NewFromFloat(1000000),
    TickSize:              decimal.NewFromFloat(0.1),
    LotSize:               decimal.NewFromFloat(0.001),
    InitialMarginRate:     decimal.NewFromFloat(0.02),    // 1/50
    MaintenanceMarginRate: decimal.NewFromFloat(0.005),   // 0.5%
    TakerFeeRate:          decimal.NewFromFloat(0.0005),  // 5bps
    HyperliquidAssetIndex: 0,
}
```

---

## 6. orders — 订单表

```go
type Order struct {
    BaseModel
    ClientOrderID string          `gorm:"type:varchar(64);index" json:"client_order_id"`
    UserID        uint64          `gorm:"index;not null" json:"user_id"`
    Symbol        string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
    Side          string          `gorm:"type:varchar(10);not null" json:"side"`
    Type          string          `gorm:"type:varchar(20);not null" json:"type"`
    Size          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
    Price         decimal.Decimal `gorm:"type:decimal(36,18)" json:"price"`
    Leverage      uint32          `gorm:"not null" json:"leverage"`
    Margin        decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"margin"`
    ReduceOnly    bool            `gorm:"default:false;not null" json:"reduce_only"`
    Status        string          `gorm:"type:varchar(20);index;not null" json:"status"`
    FilledSize    decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"filled_size"`
    ExecPrice     decimal.Decimal `gorm:"type:decimal(36,18)" json:"exec_price"`
    RealizedPnL   decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"realized_pnl"`
    Fee           decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"fee"`
    ErrorMessage  string          `gorm:"type:varchar(500)" json:"error_message"`

    User User `gorm:"foreignKey:UserID" json:"-"`
}
```

| 字段 | 枚举 / 说明 |
| --- | --- |
| side | `long` / `short` |
| type | `market` / `limit` (预留) / `liquidation` |
| status | `pending` / `filled` / `partially_filled` / `cancelled` / `rejected` |

索引：`(user_id, status)`, `(symbol, created_at)`

---

## 7. trades — 成交记录表

```go
type Trade struct {
    BaseModel
    OrderID     uint64          `gorm:"index;not null" json:"order_id"`
    UserID      uint64          `gorm:"index;not null" json:"user_id"`
    Symbol      string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
    Side        string          `gorm:"type:varchar(10);not null" json:"side"`
    Size        decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
    Price       decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"price"`
    Margin      decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"margin"`
    Fee         decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"fee"`
    RealizedPnL decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"realized_pnl"`
    IsLiquidation bool          `gorm:"default:false;not null" json:"is_liquidation"`

    Order Order `gorm:"foreignKey:OrderID" json:"-"`
    User  User  `gorm:"foreignKey:UserID" json:"-"`
}
```

---

## 8. positions — 持仓表

```go
type Position struct {
    BaseModel
    UserID           uint64          `gorm:"index;not null" json:"user_id"`
    Symbol           string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
    Side             string          `gorm:"type:varchar(10);not null" json:"side"`
    Size             decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
    EntryPrice       decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"entry_price"`
    MarkPrice        decimal.Decimal `gorm:"type:decimal(36,18)" json:"mark_price"`
    LiquidationPrice decimal.Decimal `gorm:"type:decimal(36,18)" json:"liquidation_price"`
    Margin           decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"margin"`
    Leverage         uint32          `gorm:"not null" json:"leverage"`
    UnrealizedPnL    decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"unrealized_pnl"`
    RealizedPnL      decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"realized_pnl"`
    Status           string          `gorm:"type:varchar(20);index;not null" json:"status"`
    Version          uint64          `gorm:"default:0;not null" json:"version"`

    User User `gorm:"foreignKey:UserID" json:"-"`
}
```

| 字段 | 枚举 |
| --- | --- |
| side | `long` / `short` |
| status | `open` / `closing` / `liquidating` / `closed` |

唯一约束：`(user_id, symbol, status)` — 每个用户每个 symbol 只允许一个 open 仓位。

---

## 9. vault_events — 链上事件表

```go
type VaultEvent struct {
    BaseModel
    TxHash      string          `gorm:"type:varchar(66);not null" json:"tx_hash"`
    LogIndex    uint64          `gorm:"not null" json:"log_index"`
    BlockNumber uint64          `gorm:"index;not null" json:"block_number"`
    EventType   string          `gorm:"type:varchar(20);not null" json:"event_type"`
    UserAddress string          `gorm:"type:varchar(42);index;not null" json:"user_address"`
    Amount      decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
    Status      string          `gorm:"type:varchar(20);not null" json:"status"`
    Processed   bool            `gorm:"default:false;not null;index" json:"processed"`
    ProcessedAt *time.Time      `json:"processed_at"`
    ErrorMsg    string          `gorm:"type:varchar(500)" json:"error_msg"`
}
```

幂等唯一约束：`UNIQUE(tx_hash, log_index)`

| 字段 | 枚举 |
| --- | --- |
| event_type | `deposit` / `withdraw` |
| status | `confirmed` / `processing` / `processed` / `failed` |

---

## 10. withdrawal_requests — 提现请求表

```go
type WithdrawalRequest struct {
    BaseModel
    UserID          uint64          `gorm:"index;not null" json:"user_id"`
    Amount          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
    Status          string          `gorm:"type:varchar(20);index;not null" json:"status"`
    TxHash          string          `gorm:"type:varchar(66)" json:"tx_hash"`
    Nonce           uint64          `gorm:"not null" json:"nonce"`
    Signature       string          `gorm:"type:varchar(132)" json:"signature"`
    Deadline        time.Time       `json:"deadline"`
    RejectionReason string          `gorm:"type:varchar(500)" json:"rejection_reason"`

    User User `gorm:"foreignKey:UserID" json:"-"`
}
```

| 字段 | 枚举 |
| --- | --- |
| status | `pending` / `approved` / `signed` / `submitted` / `confirmed` / `rejected` / `failed` |

---

## 11. hedge_tasks — 对冲任务表

```go
type HedgeTask struct {
    BaseModel
    Symbol              string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
    TriggerType         string          `gorm:"type:varchar(20);not null" json:"trigger_type"`
    InternalNetPosition decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"internal_net_position"`
    TargetHedgePosition decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"target_hedge_position"`
    CurrentHedgePosition decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"current_hedge_position"`
    Drift               decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"drift"`
    Status              string          `gorm:"type:varchar(20);index;not null" json:"status"`
    ErrorMessage        string          `gorm:"type:varchar(500)" json:"error_message"`
}
```

| 字段 | 枚举 |
| --- | --- |
| trigger_type | `trade` / `periodic` / `liquidation` / `manual` |
| status | `pending` / `executing` / `completed` / `failed` / `skipped` |

---

## 12. hedge_orders — 对冲订单表

```go
type HedgeOrder struct {
    BaseModel
    HedgeTaskID     uint64          `gorm:"index;not null" json:"hedge_task_id"`
    Symbol          string          `gorm:"type:varchar(20);not null" json:"symbol"`
    Side            string          `gorm:"type:varchar(10);not null" json:"side"`
    Size            decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
    Price           decimal.Decimal `gorm:"type:decimal(36,18)" json:"price"`
    ExternalOrderID string          `gorm:"type:varchar(128)" json:"external_order_id"`
    Status          string          `gorm:"type:varchar(20);index;not null" json:"status"`
    FilledSize      decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"filled_size"`
    FilledPrice     decimal.Decimal `gorm:"type:decimal(36,18)" json:"filled_price"`
    RetryCount      uint32          `gorm:"default:0" json:"retry_count"`
    ErrorMessage    string          `gorm:"type:varchar(500)" json:"error_message"`

    HedgeTask HedgeTask `gorm:"foreignKey:HedgeTaskID" json:"-"`
}
```

| 字段 | 枚举 |
| --- | --- |
| side | `buy` / `sell` |
| status | `pending` / `submitted` / `filled` / `partially_filled` / `failed` / `cancelled` |

---

## 13. liquidations — 清算记录表

```go
type Liquidation struct {
    BaseModel
    UserID           uint64          `gorm:"index;not null" json:"user_id"`
    PositionID       uint64          `gorm:"index;not null" json:"position_id"`
    Symbol           string          `gorm:"type:varchar(20);not null" json:"symbol"`
    Side             string          `gorm:"type:varchar(10);not null" json:"side"`
    Size             decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
    EntryPrice       decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"entry_price"`
    MarkPrice        decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"mark_price"`
    LiquidationPrice decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"liquidation_price"`
    ExecutionPrice   decimal.Decimal `gorm:"type:decimal(36,18)" json:"execution_price"`
    MarginReleased   decimal.Decimal `gorm:"type:decimal(36,18)" json:"margin_released"`
    RealizedPnL      decimal.Decimal `gorm:"type:decimal(36,18)" json:"realized_pnl"`
    InsuranceFund    decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"insurance_fund"`
    Type             string          `gorm:"type:varchar(20);not null" json:"type"`
    Status           string          `gorm:"type:varchar(20);index;not null" json:"status"`

    User     User     `gorm:"foreignKey:UserID" json:"-"`
    Position Position `gorm:"foreignKey:PositionID" json:"-"`
}
```

| 字段 | 枚举 |
| --- | --- |
| type | `partial` / `full` |
| status | `pending` / `executing` / `completed` / `failed` |

---

## 14. price_ticks — 价格快照表

```go
type PriceTick struct {
    ID        uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
    Symbol    string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
    IndexPrice decimal.Decimal `gorm:"type:decimal(36,18)" json:"index_price"`
    MarkPrice  decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"mark_price"`
    BestBid    decimal.Decimal `gorm:"type:decimal(36,18)" json:"best_bid"`
    BestAsk    decimal.Decimal `gorm:"type:decimal(36,18)" json:"best_ask"`
    Source     string          `gorm:"type:varchar(30)" json:"source"`
    CreatedAt  time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
}
```

按时间分区或定期清理，保留最近 N 天快照。

---

## 15. system_risk_snapshots — 系统风险快照表

```go
type SystemRiskSnapshot struct {
    ID                    uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
    Symbol                string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
    TotalLongPosition     decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"total_long_position"`
    TotalShortPosition    decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"total_short_position"`
    NetPosition           decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"net_position"`
    ExternalHedgePosition decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"external_hedge_position"`
    Drift                 decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"drift"`
    HedgeHealthy          bool            `gorm:"not null" json:"hedge_healthy"`
    TotalOpenInterest     decimal.Decimal `gorm:"type:decimal(36,18)" json:"total_open_interest"`
    CreatedAt             time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
}
```

---

## 16. insurance_fund — 保险基金表

```go
type InsuranceFund struct {
    BaseModel
    Symbol  string          `gorm:"type:varchar(20);uniqueIndex;not null" json:"symbol"`
    Balance decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"balance"`
    Version uint64          `gorm:"default:0;not null" json:"version"`
}
```

---

## ER 关系概览

```text
users 1---1 accounts
users 1---N orders
users 1---N trades
users 1---N positions
users 1---N ledger_entries
users 1---N withdrawal_requests
users 1---N liquidations

orders 1---N trades
positions 1---N liquidations

hedge_tasks 1---N hedge_orders

symbols ← orders.symbol
symbols ← positions.symbol
symbols ← hedge_tasks.symbol
```

---

## 索引策略总结

| 表 | 索引 | 类型 |
| --- | --- | --- |
| users | wallet_address | UNIQUE |
| accounts | user_id | UNIQUE |
| auth_nonces | nonce | UNIQUE |
| auth_nonces | wallet_address | INDEX |
| auth_nonces | expires_at | INDEX |
| vault_events | (tx_hash, log_index) | UNIQUE |
| vault_events | block_number | INDEX |
| vault_events | user_address | INDEX |
| vault_events | processed | INDEX |
| orders | (user_id, status) | COMPOSITE |
| orders | (symbol, created_at) | COMPOSITE |
| trades | user_id | INDEX |
| trades | order_id | INDEX |
| trades | (symbol, created_at) | COMPOSITE |
| positions | (user_id, symbol, status) | UNIQUE (open 状态) |
| ledger_entries | user_id | INDEX |
| ledger_entries | (reference_type, reference_id) | COMPOSITE |
| withdrawal_requests | user_id | INDEX |
| withdrawal_requests | status | INDEX |
| hedge_tasks | symbol | INDEX |
| hedge_tasks | status | INDEX |
| hedge_orders | hedge_task_id | INDEX |
| liquidations | user_id | INDEX |
| liquidations | position_id | INDEX |
| price_ticks | (symbol, created_at) | COMPOSITE |
| system_risk_snapshots | (symbol, created_at) | COMPOSITE |
