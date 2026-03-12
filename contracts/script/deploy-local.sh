#!/usr/bin/env bash
set -euo pipefail

# Anvil default account #0
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
OPERATOR_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
RPC_URL="http://127.0.0.1:8545"

echo "Deploying contracts to local Anvil..."

PRIVATE_KEY=$PRIVATE_KEY OPERATOR_ADDRESS=$OPERATOR_ADDRESS \
  forge script script/Deploy.s.sol:DeployScript \
  --rpc-url "$RPC_URL" \
  --broadcast \
  -vvv

echo ""
echo "Deployment complete. Copy the addresses above into backend/.env"
