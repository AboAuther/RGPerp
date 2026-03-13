# 数据库 Schema 设计

本文档描述当前代码库中已经落地的 GORM 模型与核心字段。数据库使用 MySQL 8.0，金额字段统一使用高精度十进制存储。

## 1. 设计原则

- 金额、数量、价格统一使用 `DECIMAL(36,18)` 级别精度
- 风控、清算、对冲相关状态全部持久化，支持回放与审计
- 链上事件与链下账务分离，链上事件通过 `vault_events` 入账
- 结算与兑付口径分离，账户既维护交易权益，也维护实际可兑付能力

## 2. 公共基础模型

```go
type BaseModel struct {
    ID        uint64
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## 3. 核心表

### 3.1 `users`

用户主表。

关键字段：

- `wallet_address`：唯一钱包地址
- `status`：`active / frozen / banned`

### 3.2 `auth_nonces`

钱包签名登录挑战。

关键字段：

- `wallet_address`
- `nonce`
- `message`
- `domain`
- `chain_id`
- `expires_at`
- `used`

### 3.3 `accounts`

账户权益与兑付口径主表。

当前字段：

```go
type Account struct {
    UserID              uint64
    AvailableBalance    decimal.Decimal
    LockedBalance       decimal.Decimal
    NetDeposits         decimal.Decimal
    SettledPnlBalance   decimal.Decimal
    UnsettledPnlBalance decimal.Decimal
    Version             uint64
}
```

字段含义：

- `available_balance`：可用于下单、承担 funding、承担风控扣减的可用余额
- `locked_balance`：当前被保证金占用的余额
- `net_deposits`：历史链上净入金口径，提现会递减
- `settled_pnl_balance`：已结算且具备兑付来源的盈利
- `unsettled_pnl_balance`：已实现但尚未进入兑付池的盈利
- `version`：乐观锁版本号

说明：

- `equity`、`withdrawable_balance`、`payout_capacity` 不直接存表，由风控口径实时计算

### 3.4 `settlement_pools`

平台兑付资金池。

```go
type SettlementPool struct {
    Asset   string
    Balance decimal.Decimal
    Version uint64
}
```

当前用于：

- 承接已结算盈利兑付能力
- 约束用户盈利提现上限

### 3.5 `ledger_entries`

账本流水。

关键字段：

- `user_id`
- `type`
- `amount`
- `balance_before`
- `balance_after`
- `reference_type`
- `reference_id`
- `description`

当前常见 `type`：

- `deposit`
- `withdraw`
- `margin_lock`
- `margin_release`
- `realized_pnl`
- `liquidation`
- `fee`
- `funding`
- `insurance`

## 4. 市场与价格

### 4.1 `symbols`

交易对配置表。

当前字段：

- `name`
- `base_asset`
- `quote_asset`
- `status`
- `max_leverage`
- `min_order_size`
- `max_position_notional`
- `tick_size`
- `lot_size`
- `initial_margin_rate`
- `maintenance_margin_rate`
- `maker_fee_rate`
- `taker_fee_rate`
- `hyperliquid_asset_index`

说明：

- 内部 canonical symbol 仍使用 `BTC-PERP / ETH-PERP / SOL-PERP`
- 用户显示层统一映射为 `BTC/USDC / ETH/USDC / SOL/USDC`

### 4.2 `price_ticks`

内部风控与行情快照表。

当前字段：

- `symbol`
- `index_price`
- `mark_price`
- `best_bid`
- `best_ask`
- `funding_rate`
- `funding_next_at`
- `source`
- `created_at`

用途：

- 风控价格输入
- 清算价格与风险状态计算
- ticker 与 funding countdown 展示
- 限价单 matcher 触发参考

系统已配置保留期清理，默认保留最近 72 小时。

## 5. 订单、成交与持仓

### 5.1 `orders`

统一订单表，同时承载 market 与 limit 订单。

当前字段：

```go
type Order struct {
    ClientOrderID   string
    UserID          uint64
    Symbol          string
    Side            string
    Type            string
    MarginMode      string
    Size            decimal.Decimal
    Price           decimal.Decimal
    LimitPrice      decimal.Decimal
    TimeInForce     string
    Leverage        uint32
    Margin          decimal.Decimal
    ReduceOnly      bool
    ReservedMargin  decimal.Decimal
    ReservedFee     decimal.Decimal
    Status          string
    FilledSize      decimal.Decimal
    ExecPrice       decimal.Decimal
    RealizedPnL     decimal.Decimal
    Fee             decimal.Decimal
    ErrorMessage    string
    CancelReason    string
    TriggeredAt     *time.Time
    ExpiresAt       *time.Time
    ParentOrderID   *uint64
    ExecutionSource string
}
```

当前枚举：

- `side`：`long / short`
- `type`：`market / limit`
- `margin_mode`：`isolated / cross`
- `time_in_force`：当前实现 `gtc`
- `status`：
  - `open`
  - `triggered`
  - `filled`
  - `canceled`
  - `expired`
  - `rejected`

说明：

- 限价单会冻结 `reserved_margin` 与 `reserved_fee`
- matcher 触发后，父限价单 `status=filled`，并关联一个真实执行子单

### 5.2 `trades`

成交记录表。

关键字段：

- `order_id`
- `user_id`
- `symbol`
- `side`
- `size`
- `price`
- `margin`
- `fee`
- `realized_pnl`
- `is_liquidation`

说明：

- 强平成交同样落在 `trades`
- 强平记录通过 `is_liquidation=true` 区分

### 5.3 `positions`

持仓表。

当前字段：

```go
type Position struct {
    UserID           uint64
    Symbol           string
    Side             string
    MarginMode       string
    Size             decimal.Decimal
    EntryPrice       decimal.Decimal
    MarkPrice        decimal.Decimal
    LiquidationPrice decimal.Decimal
    Margin           decimal.Decimal
    Leverage         uint32
    UnrealizedPnL    decimal.Decimal
    RealizedPnL      decimal.Decimal
    Status           string
    Version          uint64
}
```

当前枚举：

- `side`：`long / short`
- `margin_mode`：`isolated / cross`
- `status`：`open / closed`

重要语义：

- 当前实现允许同一用户、同一交易对下同时存在多条 open 仓位记录
- 同方向可合并
- 反方向不会自动净仓合并

## 6. 链上资金事件

### 6.1 `vault_events`

链上 Vault 事件入账表。

关键字段：

- `tx_hash`
- `log_index`
- `block_number`
- `event_type`
- `user_address`
- `amount`
- `status`
- `processed`
- `processed_at`
- `error_msg`

唯一约束：

- `UNIQUE(tx_hash, log_index)`

`event_type`：

- `deposit`
- `withdraw`

### 6.2 `withdrawal_requests`

提现请求表。

关键字段：

- `user_id`
- `amount`
- `status`
- `tx_hash`
- `nonce`
- `signature`
- `deadline`
- `rejection_reason`

当前状态流转：

- `pending`
- `signed`
- `submitted`
- `confirmed`
- `expired`
- `rejected`
- `failed`

## 7. 对冲、风险与清算

### 7.1 `hedge_tasks`

对冲任务表。

关键字段：

- `symbol`
- `trigger_type`
- `internal_net_position`
- `target_hedge_position`
- `current_hedge_position`
- `drift`
- `status`
- `error_message`

当前状态：

- `pending`
- `retrying`
- `buffered`
- `failed`
- `completed`
- `noop`
- `superseded`

说明：

- 当前任务目标只对齐系统内部净敞口
- 外部真实仓位仅用于监控，不再参与本次任务目标计算

### 7.2 `hedge_orders`

单个对冲任务对应的外部执行记录。

关键字段：

- `hedge_task_id`
- `symbol`
- `side`
- `size`
- `price`
- `external_order_id`
- `status`
- `filled_size`
- `filled_price`
- `retry_count`
- `error_message`

说明：

- `retry_count` 记录自动重试次数
- 当前实现支持失败后自动最多重试 3 次，并支持 admin 手动重试

### 7.3 `system_risk_snapshots`

按交易对记录的系统风险快照。

当前字段：

- `symbol`
- `net_position`
- `external_hedge_position`
- `drift`
- `hedge_healthy`
- `created_at`

说明：

- 该表用于监控和 admin 展示
- 不再作为新对冲任务目标计算的输入

### 7.4 `liquidations`

强平记录表。

关键字段：

- `user_id`
- `position_id`
- `symbol`
- `side`
- `size`
- `mark_price`
- `liquidation_price`
- `execution_price`
- `status`
- `insurance_fund`

说明：

- isolated 仓位按单仓权益触发清算
- cross 仓位按账户共享抵押物口径触发清算

## 8. Funding

### 8.1 `funding_events`

资金费率结算历史。

当前字段：

```go
type FundingEvent struct {
    UserID       uint64
    PositionID   uint64
    Symbol       string
    Side         string
    FundingRate  decimal.Decimal
    MarkPrice    decimal.Decimal
    Notional     decimal.Decimal
    Amount       decimal.Decimal
    SettlementAt time.Time
}
```

说明：

- `amount > 0`：用户收取资金费率
- `amount < 0`：用户支付资金费率
- 同一仓位同一结算时点通过唯一索引避免重复结算

## 9. 当前未纳入文档范围的内容

以下内容不属于当前数据库契约：

- 完整订单簿撮合深度数据结构
- WebSocket 推送状态表
- MQ 事件持久化表
- 外部对冲 venue 的私有订单回报表

这些能力当前不以数据库表为核心交付对象，因此不在本 Schema 中定义。
