# RESTful API 接口规范

## 1. 通用约定

### Base URL

```
http://localhost:8080/api/v1
```

### 鉴权

除标注"公开"的接口外，所有接口需在 Header 中携带 JWT：

```
Authorization: Bearer <token>
```

### 统一响应格式

成功：

```json
{
  "code": 0,
  "message": "ok",
  "data": { }
}
```

失败：

```json
{
  "code": 40001,
  "message": "insufficient balance",
  "data": null
}
```

### 错误码规划

| 范围 | 模块 |
| --- | --- |
| 10000-10999 | 鉴权 |
| 20000-20999 | 账户 |
| 30000-30999 | 订单 |
| 40000-40999 | 仓位 |
| 50000-50999 | 提现 |
| 60000-60999 | 风控 |
| 70000-70999 | 系统 |

### 分页

列表接口统一分页参数：

```
?page=1&page_size=20
```

响应：

```json
{
  "data": {
    "items": [],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 金额精度

所有金额字段以**字符串**传输，避免浮点精度丢失。

### 幂等

- 下单支持 `client_order_id`
- 提现支持 `request_id`
- Indexer 以 `tx_hash + log_index` 去重
- 关键写接口支持 `X-Idempotency-Key` Header

---

## 2. Auth 鉴权

### POST /auth/challenge `公开`

获取登录挑战。

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
  "data": {
    "nonce": "a1b2c3d4e5f6...",
    "message": "Welcome to PerpExchange!\n\nWallet: 0x1234...abcd\nNonce: a1b2c3d4e5f6...\nDomain: localhost\nChain ID: 31337\nTimestamp: 2026-03-12T10:00:00Z",
    "expires_at": "2026-03-12T10:05:00Z"
  }
}
```

### POST /auth/login `公开`

验证签名并返回 token。

Request:

```json
{
  "wallet_address": "0x1234...abcd",
  "signature": "0xabcdef...",
  "nonce": "a1b2c3d4e5f6..."
}
```

Response:

```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2026-03-13T10:00:00Z",
    "user": {
      "id": 1,
      "wallet_address": "0x1234...abcd",
      "status": "active"
    }
  }
}
```

### POST /auth/logout

登出并回收 token。

### GET /auth/session

获取当前会话信息。

---

## 3. Account 账户

### GET /account

获取账户概览。

Response:

```json
{
  "code": 0,
  "data": {
    "available_balance": "10000.000000",
    "locked_balance": "2000.000000",
    "equity": "12500.000000",
    "unrealized_pnl": "500.000000",
    "margin_ratio": "6.250000",
    "risk_level": "normal"
  }
}
```

### GET /account/ledger

获取账本流水。

Query: `page`, `page_size`, `type`

---

## 4. Deposit 充值

### GET /deposits/contract-info `公开`

获取合约地址等入金信息。

Response:

```json
{
  "code": 0,
  "data": {
    "vault_address": "0x...",
    "usdc_address": "0x...",
    "chain_id": 31337,
    "min_deposit": "10.000000"
  }
}
```

### GET /deposits

获取充值历史。

---

## 5. Withdrawal 提现

### POST /withdrawals

发起提现。后端校验通过后返回签名，前端持签名调用合约执行。

Request:

```json
{
  "amount": "1000.000000"
}
```

Response:

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "amount": "1000.000000",
    "status": "approved",
    "nonce": 42,
    "signature": "0xabcdef...",
    "deadline": "2026-03-12T11:00:00Z"
  }
}
```

### POST /withdrawals/:id/confirm

前端提现上链后，提交 tx_hash 供后端跟踪。

Request:

```json
{
  "tx_hash": "0xabc..."
}
```

### GET /withdrawals

获取提现历史。

---

## 6. Market 行情

### GET /markets `公开`

获取所有交易标的。

Response:

```json
{
  "code": 0,
  "data": [
    {
      "name": "BTC-PERP",
      "base_asset": "BTC",
      "quote_asset": "USDC",
      "status": "trading",
      "max_leverage": 50,
      "min_order_size": "0.001",
      "tick_size": "0.1",
      "lot_size": "0.001",
      "initial_margin_rate": "0.020000",
      "maintenance_margin_rate": "0.005000",
      "taker_fee_rate": "0.000500",
      "mark_price": "67500.500000",
      "index_price": "67498.000000",
      "change_24h": "2.35",
      "volume_24h": "125000000.00",
      "open_interest": "5000.000"
    }
  ]
}
```

### GET /markets/:symbol `公开`

获取单个标的详情。

### GET /markets/:symbol/price `公开`

获取实时价格。

Response:

```json
{
  "code": 0,
  "data": {
    "symbol": "BTC-PERP",
    "mark_price": "67500.500000",
    "index_price": "67498.000000",
    "best_bid": "67499.000000",
    "best_ask": "67501.000000",
    "timestamp": 1710244800000
  }
}
```

### GET /markets/:symbol/klines `公开`

获取 K 线数据。

Query: `interval` (1m/5m/15m/1h/4h/1d), `start_time`, `end_time`, `limit`

---

## 7. Order 订单

### POST /orders

下单。

Request:

```json
{
  "symbol": "BTC-PERP",
  "side": "long",
  "type": "market",
  "size": "0.01",
  "leverage": 10,
  "margin": "67.500000",
  "reduce_only": false,
  "client_order_id": "my-order-001"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| symbol | string | 是 | 交易标的 |
| side | string | 是 | `long` / `short` |
| type | string | 是 | `market` |
| size | string | 是 | 下单数量 |
| leverage | int | 是 | 杠杆倍数 |
| margin | string | 否 | 不填则自动计算 |
| reduce_only | bool | 否 | 默认 false |
| client_order_id | string | 否 | 客户端唯一标识 |

Response（市价单即时成交）:

```json
{
  "code": 0,
  "data": {
    "order": {
      "id": 1,
      "client_order_id": "my-order-001",
      "symbol": "BTC-PERP",
      "side": "long",
      "type": "market",
      "size": "0.010000",
      "leverage": 10,
      "margin": "67.500000",
      "status": "filled",
      "exec_price": "67500.000000",
      "filled_size": "0.010000",
      "fee": "0.337500",
      "realized_pnl": "0.000000",
      "created_at": "2026-03-12T10:00:00Z"
    },
    "position": {
      "id": 1,
      "symbol": "BTC-PERP",
      "side": "long",
      "size": "0.010000",
      "entry_price": "67500.000000",
      "mark_price": "67500.000000",
      "liquidation_price": "60862.500000",
      "margin": "67.500000",
      "leverage": 10,
      "unrealized_pnl": "0.000000",
      "status": "open"
    },
    "account": {
      "available_balance": "9932.162500",
      "locked_balance": "67.500000"
    }
  }
}
```

### GET /orders

获取订单列表。

Query: `page`, `page_size`, `symbol`, `status`, `side`

### GET /orders/:id

获取订单详情。

### DELETE /orders/:id

取消订单（限价单预留）。

---

## 8. Position 仓位

### GET /positions

获取所有持仓。

Query: `status` (默认 open)

Response:

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "symbol": "BTC-PERP",
      "side": "long",
      "size": "0.100000",
      "entry_price": "67500.000000",
      "mark_price": "68000.000000",
      "liquidation_price": "60862.500000",
      "margin": "675.000000",
      "leverage": 10,
      "unrealized_pnl": "50.000000",
      "realized_pnl": "0.000000",
      "margin_ratio": "13.500000",
      "risk_level": "normal",
      "status": "open"
    }
  ]
}
```

### GET /positions/:id

获取仓位详情。

### POST /positions/:id/close

平仓。

Request:

```json
{
  "size": "0.050000"
}
```

`size` 为空或等于持仓量时为全部平仓，否则为部分平仓。

### POST /positions/:id/margin

调整保证金。

Request:

```json
{
  "action": "add",
  "amount": "100.000000"
}
```

---

## 9. Trade 成交记录

### GET /trades

获取成交历史。

Query: `page`, `page_size`, `symbol`

---

## 10. Admin 管理

管理接口需管理员身份验证。

### GET /admin/risk/overview

系统风险概览。

Response:

```json
{
  "code": 0,
  "data": {
    "symbols": [
      {
        "symbol": "BTC-PERP",
        "total_long_position": "15.500000",
        "total_short_position": "12.300000",
        "net_position": "3.200000",
        "external_hedge_position": "3.150000",
        "drift": "0.050000",
        "hedge_healthy": true,
        "mark_price": "67500.000000"
      }
    ],
    "total_user_equity": "5000000.000000",
    "insurance_fund_balance": "50000.000000",
    "system_status": "normal"
  }
}
```

### GET /admin/hedge/status

对冲状态查询。Query: `symbol`

### GET /admin/liquidations

清算记录。Query: `page`, `page_size`

### POST /admin/markets/:symbol/pause

暂停指定 symbol 交易。

### POST /admin/markets/:symbol/resume

恢复交易。

### POST /admin/hedge/trigger

手动触发对冲。

Request:

```json
{
  "symbol": "BTC-PERP"
}
```

### GET /admin/risk/snapshots

风控快照历史。Query: `symbol`, `page`, `page_size`

---

## 11. WebSocket

### 连接

```
GET /ws?token={jwt}
```

### 客户端 → 服务端

订阅：

```json
{ "action": "subscribe", "channels": ["price:BTC-PERP", "account", "position", "order"] }
```

取消订阅：

```json
{ "action": "unsubscribe", "channels": ["price:BTC-PERP"] }
```

心跳：

```json
{ "action": "ping" }
```

### 服务端 → 客户端

价格推送：

```json
{
  "channel": "price:BTC-PERP",
  "type": "price.update",
  "data": {
    "symbol": "BTC-PERP",
    "mark_price": "67500.500000",
    "index_price": "67498.000000",
    "timestamp": 1710244800000
  }
}
```

余额变化：

```json
{
  "channel": "account",
  "type": "balance.update",
  "data": {
    "available_balance": "9932.162500",
    "locked_balance": "67.500000",
    "equity": "10000.000000",
    "unrealized_pnl": "0.837500"
  }
}
```

仓位变化：

```json
{
  "channel": "position",
  "type": "position.update",
  "data": {
    "id": 1,
    "symbol": "BTC-PERP",
    "side": "long",
    "size": "0.010000",
    "entry_price": "67500.000000",
    "mark_price": "67600.000000",
    "unrealized_pnl": "1.000000",
    "liquidation_price": "60862.500000",
    "margin_ratio": "15.000000"
  }
}
```

订单状态：

```json
{
  "channel": "order",
  "type": "order.update",
  "data": {
    "id": 1,
    "status": "filled",
    "exec_price": "67500.000000",
    "filled_size": "0.010000"
  }
}
```

系统通知：

```json
{
  "channel": "system",
  "type": "system.alert",
  "data": {
    "level": "warning",
    "message": "Hedge drift exceeds threshold for BTC-PERP",
    "timestamp": 1710244800000
  }
}
```

---

## 12. 状态机

### 订单

```
pending → filled
pending → rejected
filled → hedge_pending → hedged
filled → risk_alert
```

用户成交与外部对冲状态分离。对冲失败不回滚用户成交，但进入风险告警。

### 提现

```
pending → approved → signed → submitted → confirmed
pending → rejected
signed → expired
```

### 仓位

```
open → closing → closed
open → liquidating → closed
```

---

## 13. 错误码

| 错误码 | 含义 |
| --- | --- |
| 10001 | 参数非法 |
| 10002 | nonce 无效或已过期 |
| 10003 | 签名验证失败 |
| 10004 | 未认证 |
| 10005 | 无权限 |
| 20001 | 可用余额不足 |
| 20002 | 账户冻结 |
| 30001 | 保证金不足 |
| 30002 | 杠杆超限 |
| 30003 | symbol 不可交易 |
| 30004 | 价格源不可用 |
| 30005 | 超过最大仓位 |
| 40001 | 仓位不存在 |
| 40002 | 账户处于清算中 |
| 50001 | 提现金额超限 |
| 50002 | 存在未完成清算 |
| 60001 | 对冲偏差超限 |
| 70001 | 内部服务异常 |

---

## 14. 路由总览

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | `/api/v1/auth/challenge` | 获取登录挑战 | 否 |
| POST | `/api/v1/auth/login` | 签名登录 | 否 |
| POST | `/api/v1/auth/logout` | 登出 | 是 |
| GET | `/api/v1/auth/session` | 当前会话 | 是 |
| GET | `/api/v1/account` | 账户概览 | 是 |
| GET | `/api/v1/account/ledger` | 账本流水 | 是 |
| GET | `/api/v1/deposits` | 充值历史 | 是 |
| GET | `/api/v1/deposits/contract-info` | 合约信息 | 否 |
| POST | `/api/v1/withdrawals` | 请求提现 | 是 |
| POST | `/api/v1/withdrawals/:id/confirm` | 确认提现上链 | 是 |
| GET | `/api/v1/withdrawals` | 提现历史 | 是 |
| GET | `/api/v1/markets` | 标的列表 | 否 |
| GET | `/api/v1/markets/:symbol` | 标的详情 | 否 |
| GET | `/api/v1/markets/:symbol/price` | 实时价格 | 否 |
| GET | `/api/v1/markets/:symbol/klines` | K 线数据 | 否 |
| POST | `/api/v1/orders` | 下单 | 是 |
| GET | `/api/v1/orders` | 订单列表 | 是 |
| GET | `/api/v1/orders/:id` | 订单详情 | 是 |
| DELETE | `/api/v1/orders/:id` | 取消订单 | 是 |
| GET | `/api/v1/positions` | 持仓列表 | 是 |
| GET | `/api/v1/positions/:id` | 仓位详情 | 是 |
| POST | `/api/v1/positions/:id/close` | 平仓 | 是 |
| POST | `/api/v1/positions/:id/margin` | 调整保证金 | 是 |
| GET | `/api/v1/trades` | 成交历史 | 是 |
| GET | `/api/v1/admin/risk/overview` | 风控概览 | admin |
| GET | `/api/v1/admin/risk/snapshots` | 风控快照 | admin |
| GET | `/api/v1/admin/hedge/status` | 对冲状态 | admin |
| POST | `/api/v1/admin/hedge/trigger` | 手动对冲 | admin |
| GET | `/api/v1/admin/liquidations` | 清算记录 | admin |
| POST | `/api/v1/admin/markets/:symbol/pause` | 暂停交易 | admin |
| POST | `/api/v1/admin/markets/:symbol/resume` | 恢复交易 | admin |
| GET | `/ws` | WebSocket | token |

---

## 15. Gin Router 参考

```go
func SetupRouter(r *gin.Engine, h *handler.Handlers, mw *middleware.Middleware) {
    v1 := r.Group("/api/v1")

    auth := v1.Group("/auth")
    auth.POST("/challenge", h.Auth.Challenge)
    auth.POST("/login", h.Auth.Login)

    deposits := v1.Group("/deposits")
    deposits.GET("/contract-info", h.Deposit.ContractInfo)

    markets := v1.Group("/markets")
    markets.GET("", h.Market.List)
    markets.GET("/:symbol", h.Market.Detail)
    markets.GET("/:symbol/price", h.Market.Price)
    markets.GET("/:symbol/klines", h.Market.Klines)

    protected := v1.Group("")
    protected.Use(mw.JWTAuth())

    protected.POST("/auth/logout", h.Auth.Logout)
    protected.GET("/auth/session", h.Auth.Session)

    protected.GET("/account", h.Account.Overview)
    protected.GET("/account/ledger", h.Account.Ledger)

    protected.GET("/deposits", h.Deposit.List)

    wd := protected.Group("/withdrawals")
    wd.POST("", h.Withdrawal.Request)
    wd.POST("/:id/confirm", h.Withdrawal.Confirm)
    wd.GET("", h.Withdrawal.List)

    orders := protected.Group("/orders")
    orders.POST("", h.Order.Place)
    orders.GET("", h.Order.List)
    orders.GET("/:id", h.Order.Detail)
    orders.DELETE("/:id", h.Order.Cancel)

    positions := protected.Group("/positions")
    positions.GET("", h.Position.List)
    positions.GET("/:id", h.Position.Detail)
    positions.POST("/:id/close", h.Position.Close)
    positions.POST("/:id/margin", h.Position.AdjustMargin)

    protected.GET("/trades", h.Trade.List)

    admin := protected.Group("/admin")
    admin.Use(mw.AdminOnly())

    admin.GET("/risk/overview", h.Admin.RiskOverview)
    admin.GET("/risk/snapshots", h.Admin.RiskSnapshots)
    admin.GET("/hedge/status", h.Admin.HedgeStatus)
    admin.POST("/hedge/trigger", h.Admin.HedgeTrigger)
    admin.GET("/liquidations", h.Admin.Liquidations)
    admin.POST("/markets/:symbol/pause", h.Admin.PauseMarket)
    admin.POST("/markets/:symbol/resume", h.Admin.ResumeMarket)

    r.GET("/ws", mw.WSAuth(), h.WS.Handle)
}
```
