# 技术架构规范

## 1. 技术栈总览

| 层级 | 技术选型 | 说明 |
| --- | --- | --- |
| **前端** | React + Vite + TypeScript | 纯客户端 SPA |
| **UI 组件库** | Ant Design (antd) | 成熟、组件完整、TypeScript 友好、适合数据密集型金融界面 |
| **图表** | TradingView Lightweight Charts | K 线与价格曲线 |
| **Web3 交互** | wagmi + viem | 钱包连接、合约调用、签名 |
| **状态管理** | zustand | 轻量、TS 友好、适合中等规模 SPA |
| **数据请求** | TanStack Query (React Query) | 缓存、轮询、失效与重取 |
| **后端** | Go + Gin + GORM | 高性能 HTTP 框架 + ORM |
| **数据库** | MySQL 8.0 | 主业务存储 |
| **缓存** | Redis | 会话、价格缓存、分布式锁、速率限制 |
| **消息队列** | RabbitMQ | 服务间异步通信 |
| **合约开发** | Foundry (Solidity) | 编译、测试、部署 |
| **链上交互 (后端)** | go-ethereum | 事件监听、交易签名 |
| **容器编排** | Docker Compose | 本地一键启动 |

Ant Design 覆盖 Table / Form / Modal / Notification / Layout / Tabs / Statistic 等交易所常用组件，TypeScript 类型完整，几乎不需要手写 CSS。

## 2. 项目目录结构

```text
PerpExchange/
├── frontend/                          # React SPA
│   ├── public/
│   ├── src/
│   │   ├── components/
│   │   │   ├── layout/                # Header, Sidebar, Footer
│   │   │   ├── trading/               # OrderPanel, KlineChart, PositionTable
│   │   │   ├── account/               # BalanceCard, DepositModal, WithdrawModal
│   │   │   ├── auth/                  # ConnectWallet, LoginButton
│   │   │   └── common/                # LoadingSpinner, ErrorBoundary
│   │   ├── pages/
│   │   │   ├── TradePage.tsx
│   │   │   ├── AccountPage.tsx
│   │   │   ├── HistoryPage.tsx
│   │   │   └── AdminPage.tsx
│   │   ├── hooks/                     # useAuth, useBalance, usePosition, usePrice
│   │   ├── services/                  # API client, WebSocket client
│   │   ├── stores/                    # zustand stores
│   │   ├── types/                     # TypeScript 类型定义
│   │   ├── utils/                     # 格式化、常量、合约 ABI
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── index.html
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── package.json
│
├── backend/                           # Go 后端
│   ├── cmd/
│   │   ├── server/                    # API 服务入口
│   │   │   └── main.go
│   │   ├── indexer/                   # 链上事件监听服务入口
│   │   │   └── main.go
│   │   ├── hedger/                    # 对冲服务入口
│   │   │   └── main.go
│   │   └── liquidator/                # 清算服务入口
│   │       └── main.go
│   ├── internal/
│   │   ├── config/                    # 配置加载
│   │   ├── middleware/                # JWT 鉴权、CORS、限流、日志
│   │   ├── handler/                   # HTTP handler (Gin)
│   │   │   ├── auth.go
│   │   │   ├── account.go
│   │   │   ├── market.go
│   │   │   ├── order.go
│   │   │   ├── position.go
│   │   │   ├── withdrawal.go
│   │   │   ├── admin.go
│   │   │   └── ws.go
│   │   ├── service/                   # 业务逻辑层
│   │   │   ├── auth.go
│   │   │   ├── account.go
│   │   │   ├── order.go
│   │   │   ├── position.go
│   │   │   ├── withdrawal.go
│   │   │   └── market.go
│   │   ├── engine/                    # 交易引擎
│   │   │   ├── engine.go
│   │   │   ├── matcher.go
│   │   │   └── pnl.go
│   │   ├── risk/                      # 风控引擎
│   │   │   ├── risk.go
│   │   │   ├── margin.go
│   │   │   └── liquidation.go
│   │   ├── hedge/                     # 对冲服务
│   │   │   ├── hedger.go
│   │   │   ├── reconciler.go
│   │   │   └── circuit_breaker.go
│   │   ├── indexer/                   # 链上事件处理
│   │   │   ├── listener.go
│   │   │   └── processor.go
│   │   ├── price/                     # 价格服务
│   │   │   ├── provider.go
│   │   │   ├── hyperliquid.go
│   │   │   └── mock.go
│   │   ├── mq/                        # RabbitMQ 生产与消费
│   │   │   ├── publisher.go
│   │   │   └── consumer.go
│   │   ├── ws/                        # WebSocket 推送
│   │   │   └── hub.go
│   │   ├── model/                     # GORM 模型定义
│   │   │   ├── user.go
│   │   │   ├── account.go
│   │   │   ├── order.go
│   │   │   ├── trade.go
│   │   │   ├── position.go
│   │   │   ├── symbol.go
│   │   │   ├── vault_event.go
│   │   │   ├── withdrawal.go
│   │   │   ├── hedge.go
│   │   │   ├── liquidation.go
│   │   │   ├── ledger.go
│   │   │   └── price_tick.go
│   │   ├── repository/                # 数据访问层
│   │   └── pkg/                       # 工具包
│   │       ├── decimal/               # shopspring/decimal 封装
│   │       ├── hyperliquid/           # Hyperliquid API Client
│   │       ├── ethereum/              # go-ethereum 封装
│   │       ├── jwt/                   # JWT 工具
│   │       └── errors/                # 统一错误码
│   ├── migrations/                    # 数据库迁移脚本
│   ├── go.mod
│   └── go.sum
│
├── contracts/                         # Solidity 合约
│   ├── src/
│   │   ├── Vault.sol
│   │   └── MockUSDC.sol
│   ├── test/
│   ├── script/
│   └── foundry.toml
│
├── spec/                              # 技术规范文档
│   ├── TECH_ARCHITECTURE.md
│   ├── DATABASE_SCHEMA.md
│   └── API_SPEC.md
│
├── docker-compose.yml
├── README.md
├── ARCHITECTURE.md
└── AI_REPORT.md
```

## 3. 基础设施

### 3.1 Docker Compose

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: perp_exchange
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest

volumes:
  mysql_data:
```

### 3.2 环境变量

```bash
# backend/.env

# --- 服务 ---
SERVER_PORT=8080
GIN_MODE=debug

# --- MySQL ---
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=perp_exchange

# --- Redis ---
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# --- RabbitMQ ---
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# --- JWT ---
JWT_SECRET=replace_me_with_a_real_secret
JWT_EXPIRE_HOURS=24

# --- 链上 ---
RPC_URL=http://127.0.0.1:8545
CHAIN_ID=31337
VAULT_ADDRESS=0x...
USDC_ADDRESS=0x...
OPERATOR_PRIVATE_KEY=0x...

# --- Hyperliquid ---
HYPERLIQUID_API_URL=https://api.hyperliquid-testnet.xyz
HYPERLIQUID_WALLET_ADDRESS=0x...
HYPERLIQUID_PRIVATE_KEY=0x...

# --- 价格 ---
PRICE_SOURCE=hyperliquid
```

```bash
# frontend/.env.local

VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/ws
VITE_CHAIN_ID=31337
VITE_VAULT_ADDRESS=0x...
VITE_USDC_ADDRESS=0x...
```

## 4. Go 核心依赖

```text
github.com/gin-gonic/gin           # HTTP 框架
gorm.io/gorm                       # ORM
gorm.io/driver/mysql                # MySQL 驱动
github.com/redis/go-redis/v9        # Redis 客户端
github.com/rabbitmq/amqp091-go      # RabbitMQ 客户端
github.com/golang-jwt/jwt/v5        # JWT
github.com/shopspring/decimal        # 高精度十进制运算
github.com/ethereum/go-ethereum      # 以太坊交互
github.com/gorilla/websocket         # WebSocket
github.com/spf13/viper               # 配置管理
go.uber.org/zap                      # 日志
```

## 5. 前端核心依赖

```text
react                               # UI 库
react-dom
react-router-dom                    # 路由
typescript                          # 类型
vite                                # 构建
antd                                # UI 组件库
@ant-design/icons                   # 图标
@ant-design/charts                  # 图表 (可选)
lightweight-charts                  # TradingView K 线
wagmi                               # Web3 钱包
viem                                # 链上交互
@tanstack/react-query               # 数据请求
zustand                             # 状态管理
axios                               # HTTP 客户端
dayjs                               # 时间处理
```

## 6. 消息队列设计

### 6.1 Exchange 与 Queue 清单

| Exchange | Type | 绑定 Queue | 生产者 | 消费者 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `exchange.vault` | topic | `q.vault.events` | Indexer | Account Service | 链上存取款事件 |
| `exchange.trade` | topic | `q.trade.executed` | Trade Engine | Hedger, Account | 成交后触发对冲和账户更新 |
| `exchange.hedge` | topic | `q.hedge.tasks` | Trade Engine | Hedger | 对冲任务 |
| `exchange.risk` | topic | `q.risk.alerts` | Risk Engine | Admin / Alerting | 风控预警 |
| `exchange.liquidation` | topic | `q.liquidation.tasks` | Risk Engine | Liquidation Service | 清算任务 |
| `exchange.price` | fanout | `q.price.updates` | Price Service | WS Hub, Risk Engine | 价格广播 |

### 6.2 消息格式约定

所有消息体使用 JSON，包含以下公共字段：

```json
{
  "id": "uuid",
  "type": "trade.executed",
  "timestamp": 1710000000000,
  "payload": { }
}
```

## 7. Redis 使用规划

| 用途 | Key Pattern | 说明 |
| --- | --- | --- |
| 登录 nonce | `auth:nonce:{address}` | 5 分钟 TTL |
| JWT 黑名单 | `auth:blacklist:{token_hash}` | 与 token 过期时间一致 |
| 价格缓存 | `price:{symbol}:latest` | 实时 mark price |
| K 线缓存 | `price:{symbol}:kline:{interval}` | 最近 N 根 K 线 |
| 分布式锁 | `lock:order:{user_id}` | 防并发下单 |
| 分布式锁 | `lock:hedge:{symbol}` | 防并发对冲 |
| 分布式锁 | `lock:withdraw:{user_id}` | 防并发提现 |
| API 限流 | `ratelimit:{address}:{endpoint}` | 滑窗计数器 |
| WS 会话 | `ws:session:{user_id}` | 在线状态 |

## 8. WebSocket 推送设计

### 连接地址

```
ws://localhost:8080/ws?token={jwt}
```

### 频道与消息类型

| 频道 | 消息类型 | 说明 |
| --- | --- | --- |
| `price` | `price.update` | 实时价格推送 |
| `account` | `balance.update` | 余额变化 |
| `position` | `position.update` | 仓位变化 |
| `order` | `order.update` | 订单状态变化 |
| `system` | `system.alert` | 风控/维护通知 |

客户端订阅消息格式：

```json
{ "action": "subscribe", "channels": ["price:BTC-PERP", "account", "position"] }
```

服务端推送消息格式：

```json
{
  "channel": "price:BTC-PERP",
  "type": "price.update",
  "data": {
    "symbol": "BTC-PERP",
    "mark_price": "67500.50",
    "index_price": "67498.00",
    "timestamp": 1710000000000
  }
}
```

## 9. 安全设计

### 9.1 鉴权

- 钱包签名验证 + JWT Token
- JWT 过期 + Redis 黑名单支持主动登出
- 所有写操作 API 需要 JWT，读操作中敏感数据也需要

### 9.2 防重放

- 登录 challenge 包含 nonce + timestamp + domain + chainId
- nonce 单次有效，消费后作废
- 提现 nonce 上链，合约校验不可重用

### 9.3 并发保护

- 下单、提现、清算等关键路径使用 Redis 分布式锁
- 数据库关键更新使用乐观锁（GORM `version` 字段）

### 9.4 API 限流

- 基于 IP + 钱包地址的双维度限流
- 使用 Redis 滑窗计数器实现
