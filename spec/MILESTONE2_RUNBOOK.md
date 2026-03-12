# 里程碑 2 本地运行手册

本文档说明如何在本地完整运行 RGPerp 里程碑 2 的资金闭环，包括：

- 钱包签名登录
- 本地链部署 `MockUSDC` / `Vault`
- 充值入金
- Indexer 事件入账
- 提现授权
- 链上提现确认

本文档默认运行环境：

- macOS / Linux
- Docker 已安装
- Foundry 已安装，包含 `anvil`、`forge`、`cast`
- Go、Node.js、pnpm 已安装

## 1. 本地组件说明

里程碑 2 依赖以下组件：

- `MySQL`
- `Redis`
- `RabbitMQ`
- `Anvil` 本地 EVM 链
- `MockUSDC` 合约
- `Vault` 合约
- `backend server`
- `backend indexer`

说明：

- 本地联调时，`Vault` 和 `MockUSDC` 部署在 `Anvil`
- Hyperliquid 不参与里程碑 2 的充值、提现和链上事件闭环
- 对冲与真实行情接入属于后续里程碑

## 2. 本地测试账户

本地链使用 `Anvil` 默认测试账户。

当前里程碑 2 联调默认使用：

- 部署者 / operator 账户：
  - 地址：`0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266`
  - 私钥：`0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80`
- 用户账户：
  - 地址：`0x70997970C51812dc3A010C7d01b50e0d17dc79C8`
  - 私钥：`0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d`

注意：

- 这些账户仅用于本地开发链
- 不可用于真实网络
- 每次重启 `anvil`，默认账户仍然相同

## 3. 启动顺序

建议按以下顺序启动。

### 3.1 启动基础依赖

```bash
cd /Users/xiaobao/PerpExchange
docker compose up -d mysql redis rabbitmq
```

检查容器状态：

```bash
docker compose ps
```

预期：

- `rgperp-mysql` 为 `healthy`
- `rgperp-redis` 为 `healthy`
- `rgperp-rabbitmq` 为 `healthy`

### 3.2 启动本地链

新开一个终端：

```bash
cd /Users/xiaobao/PerpExchange
anvil
```

预期：

- RPC 地址：`http://127.0.0.1:8545`
- 链 ID：`31337`
- 终端输出 10 个默认账户和私钥

这个终端不要关闭。

### 3.3 部署合约

新开一个终端：

```bash
cd /Users/xiaobao/PerpExchange/contracts
bash script/deploy-local.sh
```

预期输出类似：

```text
MockUSDC deployed at: 0x5FbDB2315678afecb367f032d93F642f64180aa3
Vault deployed at: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
```

### 3.4 配置本地环境变量

确认以下文件中的地址与本次部署输出一致：

- [backend/.env](/Users/xiaobao/PerpExchange/backend/.env)
- [frontend/.env.local](/Users/xiaobao/PerpExchange/frontend/.env.local)

关键字段：

`backend/.env`

```env
SERVER_PORT=18080
DB_NAME=rg_perp
RPC_URL=http://127.0.0.1:8545
CHAIN_ID=31337
VAULT_ADDRESS=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
USDC_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
OPERATOR_PRIVATE_KEY=0xac0974...
```

`frontend/.env.local`

```env
VITE_API_BASE_URL=http://localhost:18080/api/v1
VITE_CHAIN_ID=31337
VITE_VAULT_ADDRESS=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
VITE_USDC_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
```

### 3.5 启动后端 API

新开一个终端：

```bash
cd /Users/xiaobao/PerpExchange/backend
go run cmd/server/main.go
```

健康检查：

```bash
curl -s http://127.0.0.1:18080/health
```

预期：

```json
{"code":0,"message":"ok","data":{"mysql":"ok","redis":"ok","service":"ok"}}
```

### 3.6 启动 Indexer

新开一个终端：

```bash
cd /Users/xiaobao/PerpExchange/backend
go run cmd/indexer/main.go
```

预期日志包含：

```text
indexer process initialized
```

### 3.7 启动前端

新开一个终端：

```bash
cd /Users/xiaobao/PerpExchange/frontend
pnpm dev
```

浏览器访问前端 dev server 地址。

## 4. 一键验证脚本

可以直接执行：

```bash
cd /Users/xiaobao/PerpExchange
bash .local/milestone2_flow.sh
```

该脚本会自动执行：

1. `health` 检查
2. `auth/challenge`
3. 本地钱包签名
4. `auth/login`
5. 查询初始账户
6. `mint`
7. `approve`
8. `deposit`
9. 等待 indexer 入账
10. 查询充值后账户
11. 创建提现授权
12. 调用链上 `withdraw`
13. 等待 indexer 确认
14. 查询提现后账户与提现记录

预期结果：

- 充值前 `available_balance = 0`
- 充值后 `available_balance = 100`
- 提现 `10 USDC` 后 `available_balance = 90`
- 提现记录状态为 `confirmed`

## 5. Postman 使用说明

导入文件：

- [.local/RGPerp.postman_collection.json](/Users/xiaobao/PerpExchange/.local/RGPerp.postman_collection.json)

导入后可以测试：

- `Auth Challenge`
- `Auth Login`
- `Get Account`
- `Deposit Info`
- `Create Withdrawal`
- `List Withdrawals`

说明：

- Postman 不会自动帮你用本地私钥签名 challenge message
- 推荐流程：
  1. 在 Postman 调 `Auth Challenge`
  2. 复制返回的 `message`
  3. 在终端中执行：

```bash
cast wallet sign --private-key 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d $'完整message'
```

  4. 把签名结果填回 Postman 的 `Auth Login`
  5. 拿到 `token` 后放入 collection 变量

## 6. 如何查看本地链交易和事件

### 6.1 查看 `anvil` 终端

`anvil` 终端会持续打印新区块和交易，是最直接的观察入口。

### 6.2 查看 `cast send` 返回值

每次执行链上交易，例如：

```bash
cast send <contract> "<method>" ...
```

都会返回：

- `transactionHash`
- `blockNumber`
- `status`
- `logs`

重点看：

- `status = 1` 表示交易成功
- `logs` 中可看到 ERC20 `Transfer` 与 Vault 自定义事件

### 6.3 查看交易收据

```bash
cast receipt <tx_hash> --rpc-url http://127.0.0.1:8545
```

### 6.4 查看 Vault 事件

```bash
cast logs --address 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512 --rpc-url http://127.0.0.1:8545
```

## 7. 如何查看后端服务日志

当前默认方式是直接看终端：

- API 服务日志：启动 `go run cmd/server/main.go` 的终端
- Indexer 日志：启动 `go run cmd/indexer/main.go` 的终端

你会看到：

- 数据库连接
- 自动迁移
- 路由注册
- indexer 初始化
- 错误日志

## 8. 典型排查

### 8.1 `Unknown database 'rg_perp'`

执行：

```bash
docker exec rgperp-mysql mysql -uroot -proot -e "CREATE DATABASE IF NOT EXISTS rg_perp CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

### 8.2 `listen tcp :18080: bind: address already in use`

说明本地 `18080` 被占用。

处理方式：

- 修改 [backend/.env](/Users/xiaobao/PerpExchange/backend/.env) 中的 `SERVER_PORT`
- 同步修改 [frontend/.env.local](/Users/xiaobao/PerpExchange/frontend/.env.local) 的 `VITE_API_BASE_URL`

### 8.3 登录失败 `signature verification failed`

检查：

- 使用的 message 是否和 challenge 返回完全一致
- nonce 是否已被消费
- 签名对应的钱包地址是否正确

### 8.4 充值成功但账户没变

检查：

- `indexer` 是否启动
- `RPC_URL` 是否正确
- `VAULT_ADDRESS` 是否和当前部署地址一致
- 等待 3 到 6 秒再查询账户

### 8.5 提现签名已生成，但链上 withdraw 失败

检查：

- amount 是否换算成 6 位 USDC 精度
- deadline 是否过期
- nonce 是否重复
- `OPERATOR_PRIVATE_KEY` 是否与部署时的 operator 对应
