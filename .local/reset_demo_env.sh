#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTRACTS_DIR="$ROOT_DIR/contracts"
BACKEND_ENV="$ROOT_DIR/backend/.env"
FRONTEND_ENV="$ROOT_DIR/frontend/.env.local"
LOG_DIR="$ROOT_DIR/.local/logs"
ACCOUNTS_FILE="$ROOT_DIR/.local/demo_accounts.md"

ANVIL_PID_FILE="$LOG_DIR/anvil.pid"
ANVIL_LOG_FILE="$LOG_DIR/anvil.log"

MYSQL_CONTAINER="rgperp-mysql"
REDIS_CONTAINER="rgperp-redis"
RABBITMQ_CONTAINER="rgperp-rabbitmq"

ANVIL_HOST="127.0.0.1"
ANVIL_PORT="8545"
ANVIL_RPC_URL="http://${ANVIL_HOST}:${ANVIL_PORT}"

DEPLOYER_PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
OPERATOR_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

ADMIN_WALLET="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
ADMIN_WALLET_PRIVATE_KEY="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
COMMON_USER_WALLET="0x27B0646f5813974B145e8452d2a2995D136A8622"
USDC_MINT_AMOUNT_UNITS="10000000000"
GAS_TOPUP_PER_USER="100ether"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

stop_pattern_if_running() {
  local pattern="$1"
  if pgrep -f "$pattern" >/dev/null 2>&1; then
    echo "Stopping process pattern: $pattern"
    pkill -f "$pattern" || true
    sleep 1
  fi
}

wait_for_container_healthy() {
  local container="$1"
  local attempts=60
  local status=""
  for _ in $(seq 1 "$attempts"); do
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || true)"
    if [[ "$status" == "healthy" || "$status" == "running" ]]; then
      echo "Container ready: $container ($status)"
      return 0
    fi
    sleep 2
  done

  echo "Container not healthy in time: $container (last status: $status)" >&2
  exit 1
}

update_env_value() {
  local file="$1"
  local key="$2"
  local value="$3"
  python3 - "$file" "$key" "$value" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
key = sys.argv[2]
value = sys.argv[3]

text = path.read_text()
pattern = re.compile(rf"^{re.escape(key)}=.*$", re.MULTILINE)
replacement = f"{key}={value}"
if pattern.search(text):
    text = pattern.sub(replacement, text)
else:
    if text and not text.endswith("\n"):
        text += "\n"
    text += replacement + "\n"
path.write_text(text)
PY
}

extract_address() {
  local label="$1"
  local output="$2"
  python3 - "$label" "$output" <<'PY'
import re
import sys

label = sys.argv[1]
output = sys.argv[2]
match = re.search(rf"{re.escape(label)}\s*(0x[a-fA-F0-9]{{40}})", output)
if not match:
    sys.exit(1)
print(match.group(1))
PY
}

require_command docker
require_command anvil
require_command forge
require_command cast
require_command python3

mkdir -p "$LOG_DIR"

echo "== Stopping local demo processes =="
stop_pattern_if_running "go run cmd/server/main.go"
stop_pattern_if_running "go run cmd/indexer/main.go"
stop_pattern_if_running "go run cmd/hedger/main.go"
stop_pattern_if_running "go run cmd/liquidator/main.go"
stop_pattern_if_running "go run cmd/fundingd/main.go"
stop_pattern_if_running "go run cmd/matcher/main.go"
stop_pattern_if_running "anvil"

if [[ -f "$ANVIL_PID_FILE" ]]; then
  rm -f "$ANVIL_PID_FILE"
fi

echo "== Resetting dockerized infrastructure =="
cd "$ROOT_DIR"
docker compose down -v
docker compose up -d mysql redis rabbitmq

wait_for_container_healthy "$MYSQL_CONTAINER"
wait_for_container_healthy "$REDIS_CONTAINER"
wait_for_container_healthy "$RABBITMQ_CONTAINER"

echo "== Starting fresh Anvil chain =="
nohup anvil --host "$ANVIL_HOST" --port "$ANVIL_PORT" >"$ANVIL_LOG_FILE" 2>&1 &
ANVIL_PID=$!
echo "$ANVIL_PID" > "$ANVIL_PID_FILE"
sleep 3
if ! kill -0 "$ANVIL_PID" >/dev/null 2>&1; then
  echo "Anvil failed to start. Check $ANVIL_LOG_FILE" >&2
  exit 1
fi

echo "== Deploying local contracts =="
cd "$CONTRACTS_DIR"
DEPLOY_OUTPUT="$(PRIVATE_KEY="$DEPLOYER_PRIVATE_KEY" OPERATOR_ADDRESS="$OPERATOR_ADDRESS" bash "$CONTRACTS_DIR/script/deploy-local.sh" 2>&1)"
printf '%s\n' "$DEPLOY_OUTPUT"

USDC_ADDRESS="$(extract_address 'MockUSDC deployed at:' "$DEPLOY_OUTPUT")"
VAULT_ADDRESS="$(extract_address 'Vault deployed at:' "$DEPLOY_OUTPUT")"

echo "== Updating environment files =="
update_env_value "$BACKEND_ENV" "RPC_URL" "$ANVIL_RPC_URL"
update_env_value "$BACKEND_ENV" "CHAIN_ID" "31337"
update_env_value "$BACKEND_ENV" "USDC_ADDRESS" "$USDC_ADDRESS"
update_env_value "$BACKEND_ENV" "VAULT_ADDRESS" "$VAULT_ADDRESS"
update_env_value "$BACKEND_ENV" "OPERATOR_PRIVATE_KEY" "$DEPLOYER_PRIVATE_KEY"
update_env_value "$BACKEND_ENV" "ADMIN_WALLETS" "$ADMIN_WALLET"

update_env_value "$FRONTEND_ENV" "VITE_CHAIN_ID" "31337"
update_env_value "$FRONTEND_ENV" "VITE_USDC_ADDRESS" "$USDC_ADDRESS"
update_env_value "$FRONTEND_ENV" "VITE_VAULT_ADDRESS" "$VAULT_ADDRESS"
update_env_value "$FRONTEND_ENV" "VITE_ADMIN_WALLETS" "$ADMIN_WALLET"

echo "== Funding demo wallets =="
cast send "$ADMIN_WALLET" \
  --value "$GAS_TOPUP_PER_USER" \
  --private-key "$DEPLOYER_PRIVATE_KEY" \
  --rpc-url "$ANVIL_RPC_URL" >/dev/null

cast send "$COMMON_USER_WALLET" \
  --value "$GAS_TOPUP_PER_USER" \
  --private-key "$DEPLOYER_PRIVATE_KEY" \
  --rpc-url "$ANVIL_RPC_URL" >/dev/null

cast send "$USDC_ADDRESS" \
  "mint(address,uint256)" \
  "$ADMIN_WALLET" \
  "$USDC_MINT_AMOUNT_UNITS" \
  --private-key "$DEPLOYER_PRIVATE_KEY" \
  --rpc-url "$ANVIL_RPC_URL" >/dev/null

cast send "$USDC_ADDRESS" \
  "mint(address,uint256)" \
  "$COMMON_USER_WALLET" \
  "$USDC_MINT_AMOUNT_UNITS" \
  --private-key "$DEPLOYER_PRIVATE_KEY" \
  --rpc-url "$ANVIL_RPC_URL" >/dev/null

echo "== Writing local demo account file =="
cat > "$ACCOUNTS_FILE" <<EOF
# Demo Accounts

链重置时间：$(date -u +"%Y-%m-%dT%H:%M:%SZ")
RPC_URL: $ANVIL_RPC_URL
CHAIN_ID: 31337
Mnemonic: test test test test test test test test test test test junk

## Contracts

- MockUSDC: $USDC_ADDRESS
- Vault: $VAULT_ADDRESS

## Admin Wallet

- Address: $ADMIN_WALLET
- Private Key: $ADMIN_WALLET_PRIVATE_KEY
- Role: admin
- Pre-funded ETH topup: $GAS_TOPUP_PER_USER
- Pre-minted USDC: 10000

## Common Wallet (external/custom)

- Address: $COMMON_USER_WALLET
- Private Key: not managed by local Anvil defaults
- Role: normal user
- Pre-funded ETH topup: $GAS_TOPUP_PER_USER
- Pre-minted USDC: 10000

## Default Anvil Accounts

1. $OPERATOR_ADDRESS
   - Private Key: $DEPLOYER_PRIVATE_KEY
2. $ADMIN_WALLET
   - Private Key: $ADMIN_WALLET_PRIVATE_KEY
3. 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
   - Private Key: 0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a
4. 0x90F79bf6EB2c4f870365E785982E1f101E93b906
   - Private Key: 0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6
5. 0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65
   - Private Key: 0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a
EOF

echo ""
echo "Reset completed."
echo "Anvil PID: $ANVIL_PID"
echo "Anvil log: $ANVIL_LOG_FILE"
echo "USDC_ADDRESS=$USDC_ADDRESS"
echo "VAULT_ADDRESS=$VAULT_ADDRESS"
echo "ADMIN_WALLET=$ADMIN_WALLET"
echo "COMMON_USER_WALLET=$COMMON_USER_WALLET"
echo "Accounts file: $ACCOUNTS_FILE"
echo ""
echo "Next step: restart backend services for full demo."
