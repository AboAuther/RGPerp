# RGPerp

链上托管、链下交易、外部对冲的永续合约交易系统。

系统覆盖完整交易闭环：钱包登录 → 充值 → 开仓 / 加仓 / 减仓 / 平仓 / 反手 → 账户风险计算 → mock 对冲任务生成 → 提现。

## 技术栈

| 层级 | 选型 |
| --- | --- |
| 前端 | React + Vite + TypeScript + Ant Design |
| 图表 | TradingView Advanced Chart Embed |
| Web3 | wagmi + viem |
| 状态管理 | zustand + TanStack Query |
| 后端 | Go + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis |
| 消息队列 | RabbitMQ |
| 合约 | Solidity + Foundry |
| 容器 | Docker Compose |

## 快速启动

### 1. 启动基础设施

```bash
docker compose up -d mysql redis rabbitmq
```

### 2. 启动本地链

推荐使用 `Anvil` 作为本地开发链：

```bash
anvil
```

说明：

- Vault、MockUSDC、Indexer、提现签名联调都需要本地 EVM 链
- Hyperliquid Testnet 仅用于价格联调与对冲，不替代本地 Vault 开发链

### 3. 配置环境变量

复制并填写 `backend/.env`、`frontend/.env.local`、`contracts/.env`。

可参考：

- `backend/.env.example`
- `frontend/.env.example`
- `contracts/.env.example`

### 4. 部署合约

```bash
cd contracts && forge script script/Deploy.s.sol --broadcast --rpc-url http://127.0.0.1:8545
```

### 5. 启动后端

```bash
cd backend && go run cmd/server/main.go
```

### 6. 启动链下服务

```bash
cd backend && go run cmd/indexer/main.go
cd backend && go run cmd/hedger/main.go
cd backend && go run cmd/liquidator/main.go
```

### 7. 启动前端

```bash
cd frontend && pnpm install && pnpm dev
```

## 系统架构

```mermaid
flowchart TD
    U[User]
    FE[Frontend]
    API[Backend API]
    DB[(MySQL)]
    RD[(Redis)]
    MQ[[RabbitMQ]]
    ENG[Trade Engine]
    RISK[Risk Engine]
    LIQ[Liquidation Service]
    IDX[Blockchain Indexer]
    HEDGER[Hedging Bot]
    HL[Hyperliquid Testnet]
    VAULT[Vault Contract]
    PRICE[Price Service]

    U --> FE
    FE --> API
    FE --> VAULT

    VAULT --> IDX
    IDX --> MQ
    MQ --> API
    API --> DB
    API --> RD
    API --> ENG
    API --> RISK

    ENG --> DB
    ENG --> PRICE
    ENG --> MQ

    MQ --> HEDGER
    MQ --> LIQ

    RISK --> DB
    RISK --> MQ

    HEDGER --> HL
    HEDGER --> DB

    PRICE --> HL
```

系统采用链上托管 + 链下交易 + 外部对冲的三层架构。Vault 合约负责资金托管；交易引擎、风控、清算在链下执行；净敞口对冲当前以 mock adapter 演示，后续切换至 Hyperliquid。

## 关键设计决策

**交易模型** — 采用 CFD 模式。用户订单与平台资金池即时成交，平台净风险敞口后续通过外部对冲转移。数据模型按多 symbol 设计，当前已支持 `BTC/USDC`、`ETH/USDC`、`SOL/USDC`。

**价格源** — 当前头部交易价格使用 Binance USDⓈ-M `premiumIndex`，输出 `mark_price`、`index_price`、`funding_rate` 与 `funding_next_at`；图表使用 TradingView 的 Binance 永续市场数据。系统同时保留 `mock`、`hyperliquid` 与链上 `oracle` 价格源接口，便于后续切换。

**对冲策略** — 按交易所净敞口统一对冲，而非逐笔用户订单对冲。当前已实现 hedge task 生成、hedger 进程和 mock adapter；Hyperliquid adapter 已完成结构拆分，真实签名与正式下单作为下一阶段重点。

**提现模型** — 后端签名授权提现。用户发起提现请求后，后端校验可用余额、仓位风险和风控状态，生成签名后由用户调用合约执行。

**清算机制** — 当前已实现账户权益、维持保证金、风险率与提现前风险校验；独立 Liquidation Service 与强平执行链路属于下一阶段主线。

**风控体系** — 覆盖交易前（保证金、杠杆、限仓、价格有效性）、持仓中（风险率、维持保证金、对冲偏差）、提现前（余额、冻结状态、限额）三个维度。

## 已知约束

- 当前已支持 `BTC/USDC`、`ETH/USDC`、`SOL/USDC` 三个交易对展示与交易，更多 symbol 可按 schema 继续扩展
- 当前订单类型为 Market Order，限价单、撤单与条件单作为后续迭代
- Hyperliquid 真实签名下单尚未接通，当前对冲执行为 mock adapter
- 独立清算服务、保险基金动用、自动减仓（ADL）仍属后续阶段

## 文档索引

| 文档 | 内容 |
| --- | --- |
| `ARCHITECTURE.md` | 系统架构、模块划分、数据流、风控与清算设计 |
| `spec/TECH_ARCHITECTURE.md` | 技术栈、目录结构、基础设施、消息队列、安全设计 |
| `spec/DATABASE_SCHEMA.md` | 完整 GORM 表结构与索引策略 |
| `spec/API_SPEC.md` | RESTful API 接口规范与 WebSocket 协议 |
| `AI_REPORT.md` | AI 工具使用报告 |
