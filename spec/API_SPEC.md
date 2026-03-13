# REST API 接口规范

本文档描述当前代码库已实现的 HTTP API 契约。未在本文档中列出的接口，不视为当前交付范围。

## 1. 通用约定

### Base URL

本地默认地址：

```text
http://127.0.0.1:18080/api/v1
```

### 鉴权

除公开接口外，所有接口均要求：

```text
Authorization: Bearer <jwt>
```

### 统一响应

成功：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

失败：

```json
{
  "code": 30010,
  "message": "insufficient margin",
  "data": null
}
```

### 金额与精度

- 所有金额、数量、价格字段均按字符串语义处理
- 后端使用 `decimal.Decimal`
- 前端显示层可按业务场景格式化

### 分页与查询

当前列表接口统一采用 `limit` 参数：

```text
?limit=20
```

默认 `20`，最大 `100`。

## 2. Auth

### `POST /auth/challenge` 公开

请求钱包登录挑战消息。

Request:

```json
{
  "wallet_address": "0x1234...abcd"
}
```

Response:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "nonce": "a1b2c3...",
    "message": "Welcome to RGPerp!\n\nWallet: 0x1234...abcd\nNonce: a1b2c3...",
    "expires_at": "2026-03-13T10:05:00Z"
  }
}
```

### `POST /auth/login` 公开

提交签名并换取 JWT。

Request:

```json
{
  "wallet_address": "0x1234...abcd",
  "signature": "0x...",
  "nonce": "a1b2c3..."
}
```

Response:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2026-03-14T10:00:00Z",
    "user": {
      "id": 1,
      "wallet_address": "0x1234...abcd",
      "status": "active"
    }
  }
}
```

## 3. Market

### `GET /markets` 公开

返回当前可交易标的配置。

Response item:

```json
{
  "name": "BTC-PERP",
  "base_asset": "BTC",
  "quote_asset": "USDC",
  "status": "trading",
  "max_leverage": 50,
  "min_order_size": "0.001",
  "tick_size": "0.1",
  "lot_size": "0.001",
  "initial_margin_rate": "0.02",
  "maintenance_margin_rate": "0.005",
  "maker_fee_rate": "0",
  "taker_fee_rate": "0.0005"
}
```

### `GET /markets/:symbol/price` 公开

返回交易对最新 ticker，结构与 `/ticker` 一致。

### `GET /markets/:symbol/ticker` 公开

Response:

```json
{
  "symbol": "BTC-PERP",
  "display_pair": "BTC/USDC",
  "base_asset": "BTC",
  "quote_asset": "USDC",
  "mark_price": "72400.1",
  "index_price": "72398.5",
  "best_bid": "72399.9",
  "best_ask": "72400.3",
  "source": "binance",
  "timestamp": 1773391200,
  "change_24h": "1250.6",
  "change_24h_pct": "1.75",
  "volume_24h": "2520000.12",
  "open_interest": "1834000.00",
  "funding_rate": "0.0001",
  "funding_next_at": 1773394800,
  "max_leverage": 50
}
```

### `GET /markets/:symbol/klines` 公开

Query:

- `interval`: `1m` / `5m` / `15m` / `1h` / `1d`
- `limit`
- `start_time`
- `end_time`

Response item:

```json
{
  "time": 1773391200,
  "open": "72310.1",
  "high": "72420.5",
  "low": "72290.2",
  "close": "72400.1",
  "volume": "12345.67"
}
```

## 4. Account

### `GET /account`

返回账户概览、风险口径与兑付口径。

Response:

```json
{
  "asset": "USDC",
  "available_balance": "873.241212",
  "locked_balance": "28",
  "net_deposits": "901.241212",
  "settled_pnl_balance": "0",
  "unsettled_pnl_balance": "0",
  "payout_capacity": "0",
  "pending_withdrawal": "0",
  "withdrawable_balance": "0",
  "unrealized_pnl": "0",
  "equity": "901.241212",
  "maintenance_margin": "0",
  "margin_ratio": "0",
  "risk_level": "normal"
}
```

### `GET /deposits`

返回当前登录用户充值记录。

Response:

```json
{
  "items": [
    {
      "tx_hash": "0x...",
      "log_index": 0,
      "block_number": 123,
      "amount": "100",
      "status": "confirmed",
      "created_at": "2026-03-13T10:00:00Z"
    }
  ]
}
```

### `GET /funding-history`

返回资金费率结算记录。

Query:

- `limit`

Response item:

```json
{
  "id": 1,
  "symbol": "BTC-PERP",
  "display_pair": "BTC/USDC",
  "side": "long",
  "funding_rate": "0.0001",
  "mark_price": "72400",
  "notional": "72.4",
  "amount": "-0.00724",
  "settlement_at": "2026-03-13T20:00:00Z",
  "created_at": "2026-03-13T20:00:01Z"
}
```

### `GET /wallet/deposit-info`

返回前端执行链上充值所需的链信息。

Response:

```json
{
  "chain_id": 31337,
  "vault_address": "0x...",
  "usdc_address": "0x..."
}
```

## 5. Withdrawals

### `POST /withdrawals`

创建提现请求并返回链上提现签名。

Request:

```json
{
  "amount": "10"
}
```

Response:

```json
{
  "id": 12,
  "amount": "10",
  "status": "signed",
  "nonce": "16506712804377430000",
  "signature": "0x...",
  "deadline": "2026-03-13T10:24:43Z"
}
```

### `GET /withdrawals`

返回提现请求历史。

Response item:

```json
{
  "id": 12,
  "amount": "10",
  "status": "confirmed",
  "nonce": "16506712804377430000",
  "signature": "0x...",
  "deadline": "2026-03-13T10:24:43Z",
  "tx_hash": "0x...",
  "created_at": "2026-03-13T10:14:43Z"
}
```

## 6. Orders

### `POST /orders`

统一创建市价单或限价单。

Request:

```json
{
  "client_order_id": "demo-001",
  "symbol": "BTC-PERP",
  "side": "long",
  "type": "limit",
  "margin_mode": "isolated",
  "size": "0.001",
  "limit_price": "72000",
  "time_in_force": "gtc",
  "leverage": 10,
  "margin": "7.2",
  "reduce_only": false,
  "test_mode": false
}
```

字段说明：

- `side`: `long` / `short`
- `type`: `market` / `limit`
- `margin_mode`: `isolated` / `cross`
- `time_in_force`: 当前实现 `gtc`
- `reduce_only`: 平仓 / 减仓意图
- `test_mode`: 仅测试高杠杆联调用

Response 为订单详情，核心字段包括：

```json
{
  "id": 101,
  "client_order_id": "demo-001",
  "symbol": "BTC-PERP",
  "side": "long",
  "type": "limit",
  "margin_mode": "isolated",
  "size": "0.001",
  "limit_price": "72000",
  "reserved_margin": "7.2",
  "reserved_fee": "0.036",
  "status": "open",
  "execution_source": "direct"
}
```

### `POST /orders/:id/cancel`

取消未成交限价单并释放冻结资金。

### `GET /orders`

返回订单历史，包含 market 与 limit 订单。

### `GET /open-orders`

返回当前未完成挂单，主要用于限价单展示。

### `GET /trades`

返回成交历史。

Response item:

```json
{
  "id": 88,
  "order_id": 101,
  "symbol": "BTC-PERP",
  "display_pair": "BTC/USDC",
  "side": "long",
  "size": "0.001",
  "price": "71751.8",
  "margin": "7.2",
  "fee": "0.0358759",
  "realized_pnl": "0",
  "is_liquidation": false,
  "created_at": "2026-03-13T10:00:00Z"
}
```

### `GET /positions`

返回当前 open positions。

Response item:

```json
{
  "id": 44,
  "symbol": "BTC-PERP",
  "display_pair": "BTC/USDC",
  "side": "long",
  "margin_mode": "isolated",
  "size": "0.001",
  "entry_price": "71751.8",
  "mark_price": "72400",
  "liquidation_price": "68920.4",
  "margin": "7.2",
  "leverage": 10,
  "unrealized_pnl": "0.6482",
  "realized_pnl": "0",
  "notional": "72.4",
  "maintenance_margin": "0.362",
  "risk_ratio": "0.05",
  "status": "open"
}
```

说明：

- 当前实现允许同一用户、同一交易对下存在多条 open 仓位记录
- 同方向下单可合并到同方向持仓
- 反方向下单不会净仓合并

## 7. Admin

以下接口要求：

- 已登录
- 钱包地址位于 `ADMIN_WALLETS` 白名单

### `GET /admin/overview`

返回系统级概览，包括：

- 交易中市场数
- 风险账户数
- 对冲任务状态汇总
- 按交易对展示的净敞口偏差摘要

### `GET /admin/hedge-tasks`

返回最近对冲任务历史。

Response item:

```json
{
  "id": 68,
  "created_at": "2026-03-13T22:13:39Z",
  "symbol": "BTC-PERP",
  "display_pair": "BTC/USDC",
  "trigger_type": "trade",
  "internal_net_position": "0",
  "target_hedge_position": "0",
  "current_hedge_position": "-0.03038",
  "drift": "0.03038",
  "status": "failed",
  "error_message": "Price too far from oracle",
  "external_order_status": "failed",
  "retry_count": 3
}
```

### `POST /admin/hedge-tasks/:id/retry`

手动重试失败对冲任务。

### `GET /admin/risk-snapshots`

返回按交易对记录的风险快照历史：

- `symbol`
- `internal_net_position`
- `external_hedge_position`
- `drift`
- `hedge_healthy`
- `created_at`

### `GET /admin/liquidations`

返回强平记录列表。

### `GET /admin/alerts`

返回当前风险与对冲告警列表。

## 8. 错误语义

当前前端已按后端 `message` 直接展示业务错误，常见错误包括：

- `insufficient margin`
- `price data is stale`
- `withdrawal rejected by risk check`
- `opposite-side orders must close the current position first`
- `account is in reduce-only mode`

文档中不再维护一份脱离代码的静态错误码大全，实际以 `internal/pkg/errors` 和接口响应为准。
