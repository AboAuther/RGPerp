# 清算测试 Runbook

这份文档给出一套可重复执行的清算测试流程，目标是稳定验证以下链路：

1. 用户登录
2. 链上充值，经 `Indexer` 入账
3. 用户开仓
4. `mark price` 下跌触发风险升级
5. `Liquidator` 自动清算并生成强平记录
6. 清算后仓位、账户、交易、对冲任务状态符合预期

## 1. 推荐执行方式

建议分两层执行：

1. 自动回归测试：用于快速检查核心逻辑没有回归
2. 手工完整链路：用于验证真实服务进程、API、数据库、清算器之间的联动

## 2. 自动回归测试

在项目根目录执行：

```bash
cd /Users/xiaobao/PerpExchange/backend
go test ./internal/service -run 'TestLiquidatorService_LiquidatesDangerPosition|TestOrderService_RejectsCustomMarginBelowMinimum|TestOrderService_AddingPositionRecomputesEffectiveLeverage|TestOrderService_RejectsAggregatePositionNotionalExceedingLimit|TestRiskState_UsesActualLockedMarginAsTotalInitial' -v
```

预期：

- 所有测试通过
- 覆盖以下关键修复点：
  - 自定义保证金不能低于最小初始保证金
  - 小额高杠杆加仓不能重写整仓杠杆
  - 总仓位名义价值不能突破 `MaxPositionNotional`
  - 风控 `TotalInitial` 与真实锁仓保证金一致
  - 危险仓位会被 `Liquidator` 自动平掉

若要跑完整后端测试：

```bash
cd /Users/xiaobao/PerpExchange/backend
go test ./...
```

## 3. 手工完整链路测试

### 3.1 依赖工具

- `docker`
- `anvil`
- `forge`
- `cast`
- `jq`
- `curl`
- `mysql` 客户端

### 3.2 测试账户

本地链使用 `Anvil` 默认账户：

- operator
  - 地址：`0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266`
  - 私钥：`0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80`
- 用户
  - 地址：`0x70997970C51812dc3A010C7d01b50e0d17dc79C8`
  - 私钥：`0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d`

## 4. 环境准备

### 4.1 最干净的重复执行方式

每次都从干净环境开始，最稳定：

```bash
cd /Users/xiaobao/PerpExchange
docker compose down -v
docker compose up -d mysql redis rabbitmq
```

然后重新启动本地链：

```bash
cd /Users/xiaobao/PerpExchange
anvil
```

重新部署合约：

```bash
cd /Users/xiaobao/PerpExchange/contracts
bash script/deploy-local.sh
```

### 4.2 后端环境变量建议

建议在 `backend/.env` 中至少确认以下配置：

```env
SERVER_PORT=18080
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=rg_perp
RPC_URL=http://127.0.0.1:8545
CHAIN_ID=31337
PRICE_SOURCE=mock
PRICE_POLL_INTERVAL_MS=600000
JWT_SECRET=replace_me_with_a_real_secret
JWT_EXPIRE_HOURS=24
```

说明：

- `PRICE_SOURCE=mock`：保证本地价格源可用
- `PRICE_POLL_INTERVAL_MS=600000`：把自动价格刷新拉长到 10 分钟，避免你手动插入的清算价格被立即覆盖

### 4.3 启动进程

启动 API：

```bash
cd /Users/xiaobao/PerpExchange/backend
go run cmd/server/main.go
```

启动 Indexer：

```bash
cd /Users/xiaobao/PerpExchange/backend
go run cmd/indexer/main.go
```

启动 Liquidator：

```bash
cd /Users/xiaobao/PerpExchange/backend
go run cmd/liquidator/main.go
```

健康检查：

```bash
curl -s http://127.0.0.1:18080/health | jq .
```

预期：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "mysql": "ok",
    "redis": "ok",
    "service": "ok"
  }
}
```

## 5. 一次完整清算测试

### 5.1 设置变量

```bash
export API_BASE_URL="http://127.0.0.1:18080/api/v1"
export HEALTH_URL="http://127.0.0.1:18080/health"
export RPC_URL="http://127.0.0.1:8545"
export MYSQL_HOST="127.0.0.1"
export MYSQL_PORT="3306"
export MYSQL_USER="root"
export MYSQL_PASSWORD="root"
export MYSQL_DB="rg_perp"

export USER_ADDRESS="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
export USER_PRIVATE_KEY="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"

export OPERATOR_PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
export USDC_ADDRESS="<替换成 deploy-local.sh 输出的 MockUSDC 地址>"
export VAULT_ADDRESS="<替换成 deploy-local.sh 输出的 Vault 地址>"
```

### 5.2 登录并获取 JWT

获取 challenge：

```bash
CHALLENGE_JSON="$(curl -s -X POST "$API_BASE_URL/auth/challenge" \
  -H 'Content-Type: application/json' \
  -d "{\"wallet_address\":\"$USER_ADDRESS\",\"domain\":\"localhost:18080\",\"chain_id\":31337}")"

echo "$CHALLENGE_JSON" | jq .
NONCE="$(echo "$CHALLENGE_JSON" | jq -r '.data.nonce')"
MESSAGE="$(echo "$CHALLENGE_JSON" | jq -r '.data.message')"
```

用本地私钥签名 challenge message：

```bash
SIGNATURE="$(cast wallet sign --private-key "$USER_PRIVATE_KEY" "$MESSAGE")"
echo "$SIGNATURE"
```

登录：

```bash
LOGIN_JSON="$(curl -s -X POST "$API_BASE_URL/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"wallet_address\":\"$USER_ADDRESS\",\"nonce\":\"$NONCE\",\"signature\":\"$SIGNATURE\"}")"

echo "$LOGIN_JSON" | jq .
TOKEN="$(echo "$LOGIN_JSON" | jq -r '.data.token')"
AUTH_HEADER="Authorization: Bearer $TOKEN"
```

### 5.3 给用户充值 100 USDC

铸币：

```bash
cd /Users/xiaobao/PerpExchange/contracts
bash script/mint-local.sh "$USER_ADDRESS" 1000
```

授权 Vault：

```bash
cast send "$USDC_ADDRESS" "approve(address,uint256)" "$VAULT_ADDRESS" 100000000 \
  --private-key "$USER_PRIVATE_KEY" \
  --rpc-url "$RPC_URL"
```

充值 100 USDC：

```bash
cast send "$VAULT_ADDRESS" "deposit(uint256)" 100000000 \
  --private-key "$USER_PRIVATE_KEY" \
  --rpc-url "$RPC_URL"
```

等待 `Indexer` 入账：

```bash
sleep 6
curl -s "$API_BASE_URL/account" -H "$AUTH_HEADER" | jq .
```

预期：

- `available_balance` 约为 `100`
- `locked_balance` 为 `0`
- `risk_level` 为 `normal`

### 5.4 开一个容易触发清算的多仓

这里选择：

- `BTC-PERP`
- `side = long`
- `size = 0.011`
- `leverage = 10`
- `margin_mode = isolated`

下单：

```bash
ORDER_JSON="$(curl -s -X POST "$API_BASE_URL/orders" \
  -H 'Content-Type: application/json' \
  -H "$AUTH_HEADER" \
  -d '{
    "client_order_id": "liq-manual-001",
    "symbol": "BTC-PERP",
    "side": "long",
    "type": "market",
    "margin_mode": "isolated",
    "size": "0.011",
    "leverage": 10
  }')"

echo "$ORDER_JSON" | jq .
```

查看账户与仓位：

```bash
curl -s "$API_BASE_URL/account" -H "$AUTH_HEADER" | jq .
curl -s "$API_BASE_URL/positions" -H "$AUTH_HEADER" | jq .
```

预期：

- 存在一条 `BTC-PERP` 多仓
- `locked_balance` 大于 `0`
- 仓位返回里有 `liquidation_price`

### 5.5 手动插入低价格，驱动清算

将 `mark price` 直接插到一个足够低的值，例如 `70000`。

```bash
mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DB" -e "
INSERT INTO price_ticks (
  symbol,
  index_price,
  mark_price,
  best_bid,
  best_ask,
  funding_rate,
  source,
  created_at
) VALUES (
  'BTC-PERP',
  70000,
  70000,
  69990,
  70010,
  0,
  'manual-liquidation-test',
  UTC_TIMESTAMP(6)
);"
```

说明：

- 对于单仓用户，当前实现会在 `Liquidator` 扫描到 `liquidating` 后直接全平
- 多仓时默认是 50% 部分清算；但这里只有 1 个仓位，所以会走 `full`

等待清算器扫描：

```bash
sleep 5
```

### 5.6 验证清算结果

#### 验证 1：账户状态

```bash
curl -s "$API_BASE_URL/account" -H "$AUTH_HEADER" | jq .
```

预期：

- `locked_balance` 显著下降，通常回到 `0`
- 如果该仓位已被完全平掉，`risk_level` 不应继续是 `liquidating`

#### 验证 2：仓位已被强平

```bash
curl -s "$API_BASE_URL/positions" -H "$AUTH_HEADER" | jq .
```

预期：

- 当前测试场景下应返回空数组，或至少 `BTC-PERP` 不再是原始仓位大小

#### 验证 3：最近交易中出现 liquidation trade

```bash
curl -s "$API_BASE_URL/trades?limit=10" -H "$AUTH_HEADER" | jq .
```

预期：

- 最近一条或几条 `trade` 中，存在 `is_liquidation = true`

#### 验证 4：管理端生成 liquidation 记录

```bash
curl -s "$API_BASE_URL/admin/liquidations?limit=10" -H "$AUTH_HEADER" | jq .
```

预期：

- 最新记录里有当前用户的 `BTC-PERP` 清算
- `type = "full"`
- `status = "completed"`
- `mark_price = 70000`
- 多仓执行价应接近 `70000 * (1 - 0.0025) = 69825`

#### 验证 5：生成 liquidation hedge task

```bash
curl -s "$API_BASE_URL/admin/hedge-tasks?limit=10" -H "$AUTH_HEADER" | jq .
```

预期：

- 最新 `hedge_task` 的 `trigger_type = "liquidation"`
- `symbol = "BTC-PERP"`

#### 验证 6：数据库直接校验

查看最近清算记录：

```bash
mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DB" -e "
SELECT id, user_id, symbol, side, size, mark_price, execution_price, realized_pnl, type, status, created_at
FROM liquidations
ORDER BY id DESC
LIMIT 5;"
```

查看仓位状态：

```bash
mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DB" -e "
SELECT id, user_id, symbol, side, size, margin, leverage, status, liquidation_price, updated_at
FROM positions
WHERE user_id = (SELECT id FROM users WHERE wallet_address = '$USER_ADDRESS')
ORDER BY id DESC;"
```

## 6. 本次测试的预期业务逻辑

这条测试验证的是下面这条完整链路：

1. 用户以 `10x` 开多 `BTC-PERP`
2. 系统锁定保证金并记录仓位
3. 市场价格被打到危险区间
4. `Risk Engine` 将账户识别为 `liquidating`
5. `Liquidator` 用惩罚执行价反向平仓
6. 系统写入：
   - `orders`
   - `trades`
   - `liquidations`
   - `ledger_entries`
   - `hedge_tasks`
7. 用户仓位关闭，风险状态恢复

## 7. 重复执行建议

### 7.1 最推荐

每次都重置：

1. `docker compose down -v`
2. 重启 `anvil`
3. 重新部署合约
4. 重新执行本 runbook

这是最稳定、最不容易受历史数据干扰的做法。

### 7.2 如果只想快速重跑

至少保证：

- 使用新的 `client_order_id`
- 重新充值
- 重新插入一条新的低价 `price_ticks`

## 8. 失败排查

### 8.1 一直没有发生清算

优先检查：

- `Liquidator` 进程是否真的在运行
- 新插入的 `price_ticks.created_at` 是否是最新时间
- `PRICE_POLL_INTERVAL_MS` 是否太短，导致你的手动价格被自动 mock 价格覆盖
- `/account` 返回的 `risk_level` 是否已经进入 `liquidating`

### 8.2 下单失败

优先检查：

- 充值是否已被 `Indexer` 入账
- `available_balance` 是否足够覆盖 `margin + fee`
- `BTC-PERP` 是否处于 `trading` 状态

### 8.3 看不到 liquidation 记录

优先检查：

- 是否把价格打得足够低
- 是否只等待了 1~2 秒，建议至少等待 5 秒
- `admin/liquidations` 是否带了有效 JWT

## 9. 建议后续补充

这份 runbook 当前覆盖的是“单用户、单仓位、自动全平”场景。后续建议继续补：

1. 多仓位用户的部分清算流程
2. 空头仓位的清算流程
3. 清算后保险基金扣减校验
4. 价格过期时不允许清算的保护测试
