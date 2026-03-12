#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"
USDC_ADDRESS="${USDC_ADDRESS:-0x5FbDB2315678afecb367f032d93F642f64180aa3}"
MINTER_PRIVATE_KEY="${MINTER_PRIVATE_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"

TO_ADDRESS="${1:-0x70997970C51812dc3A010C7d01b50e0d17dc79C8}"
AMOUNT_USDC="${2:-1000}"

if ! [[ "$AMOUNT_USDC" =~ ^[0-9]+$ ]]; then
  echo "AMOUNT_USDC must be an integer (e.g. 10, 100, 1000)"
  exit 1
fi

AMOUNT_UNITS=$((AMOUNT_USDC * 1000000))

echo "Minting ${AMOUNT_USDC} USDC (${AMOUNT_UNITS} units) to ${TO_ADDRESS}..."
cast send "${USDC_ADDRESS}" \
  "mint(address,uint256)" \
  "${TO_ADDRESS}" \
  "${AMOUNT_UNITS}" \
  --private-key "${MINTER_PRIVATE_KEY}" \
  --rpc-url "${RPC_URL}"

echo "Mint done."
