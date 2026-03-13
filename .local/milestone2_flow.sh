#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:18080/api/v1}"
RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"
USER_ADDRESS="${USER_ADDRESS:-0x70997970C51812dc3A010C7d01b50e0d17dc79C8}"
USER_PRIVATE_KEY="${USER_PRIVATE_KEY:-0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d}"
VAULT_ADDRESS="${VAULT_ADDRESS:-0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512}"
USDC_ADDRESS="${USDC_ADDRESS:-0x5FbDB2315678afecb367f032d93F642f64180aa3}"

echo "== health =="
curl -s "$API_BASE_URL/../health"
echo

echo "== auth challenge =="
CHALLENGE_JSON="$(curl -s -X POST "$API_BASE_URL/auth/challenge" \
  -H 'Content-Type: application/json' \
  -d "{\"wallet_address\":\"$USER_ADDRESS\",\"domain\":\"localhost:18080\",\"chain_id\":31337}")"
echo "$CHALLENGE_JSON" | jq .

NONCE="$(echo "$CHALLENGE_JSON" | jq -r '.data.nonce')"
MESSAGE="$(echo "$CHALLENGE_JSON" | jq -r '.data.message')"
SIGNATURE="$(cast wallet sign --private-key "$USER_PRIVATE_KEY" "$MESSAGE")"

echo "== auth login =="
LOGIN_JSON="$(curl -s -X POST "$API_BASE_URL/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"wallet_address\":\"$USER_ADDRESS\",\"nonce\":\"$NONCE\",\"signature\":\"$SIGNATURE\"}")"
echo "$LOGIN_JSON" | jq .

TOKEN="$(echo "$LOGIN_JSON" | jq -r '.data.token')"
AUTH_HEADER="Authorization: Bearer $TOKEN"

echo "== account before deposit =="
curl -s "$API_BASE_URL/account" -H "$AUTH_HEADER" | jq .

echo "== mint / approve / deposit =="
cast send "$USDC_ADDRESS" "mint(address,uint256)" "$USER_ADDRESS" 1000000000 \
  --private-key "$USER_PRIVATE_KEY" --rpc-url "$RPC_URL" >/dev/null
cast send "$USDC_ADDRESS" "approve(address,uint256)" "$VAULT_ADDRESS" 100000000 \
  --private-key "$USER_PRIVATE_KEY" --rpc-url "$RPC_URL" >/dev/null
cast send "$VAULT_ADDRESS" "deposit(uint256)" 100000000 \
  --private-key "$USER_PRIVATE_KEY" --rpc-url "$RPC_URL" >/dev/null

sleep 6
echo "== account after deposit =="
curl -s "$API_BASE_URL/account" -H "$AUTH_HEADER" | jq .

echo "== request withdrawal =="
WITHDRAW_JSON="$(curl -s -X POST "$API_BASE_URL/withdrawals" \
  -H 'Content-Type: application/json' \
  -H "$AUTH_HEADER" \
  -H 'X-Idempotency-Key: wd-script-001' \
  -d '{"amount":"10"}')"
echo "$WITHDRAW_JSON" | jq .

WITHDRAW_NONCE="$(echo "$WITHDRAW_JSON" | jq -r '.data.nonce')"
WITHDRAW_SIGNATURE="$(echo "$WITHDRAW_JSON" | jq -r '.data.signature')"
WITHDRAW_DEADLINE="$(echo "$WITHDRAW_JSON" | jq -r '.data.deadline')"
WITHDRAW_DEADLINE_UNIX="$(node -e "console.log(Math.floor(new Date(process.argv[1]).getTime()/1000))" "$WITHDRAW_DEADLINE")"

echo "== execute withdrawal onchain =="
cast send "$VAULT_ADDRESS" "withdraw(uint256,uint256,uint256,bytes)" 10000000 "$WITHDRAW_NONCE" "$WITHDRAW_DEADLINE_UNIX" "$WITHDRAW_SIGNATURE" \
  --private-key "$USER_PRIVATE_KEY" --rpc-url "$RPC_URL" >/dev/null

sleep 6
echo "== account after withdrawal =="
curl -s "$API_BASE_URL/account" -H "$AUTH_HEADER" | jq .

echo "== withdrawals =="
curl -s "$API_BASE_URL/withdrawals" -H "$AUTH_HEADER" | jq .
