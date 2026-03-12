#!/usr/bin/env python3
import json
import sys
from typing import Any

from eth_account import Account
from hyperliquid.exchange import Exchange
from hyperliquid.info import Info


EMPTY_SPOT_META = {"universe": [], "tokens": []}


def emit(payload: dict[str, Any]) -> None:
    sys.stdout.write(json.dumps(payload))


def parse_filled(data: dict[str, Any]) -> tuple[str, str, str]:
    statuses = (((data.get("response") or {}).get("data") or {}).get("statuses")) or []
    if not statuses:
        raise RuntimeError("missing order statuses")
    item = statuses[0]
    if "error" in item:
        raise RuntimeError(item["error"])
    filled = item.get("filled")
    if not filled:
        raise RuntimeError(f"unexpected order status: {item}")
    return str(filled.get("oid", "")), str(filled.get("totalSz", "0")), str(filled.get("avgPx", "0"))


def coin_for_symbol(symbol: str) -> str:
    return symbol.split("-")[0].split("/")[0].upper()


def main() -> None:
    payload = json.load(sys.stdin)
    action = payload["action"]
    api_url = payload["api_url"]
    private_key = payload.get("private_key") or ""
    wallet_address = payload.get("wallet_address") or ""
    symbol = payload["symbol"]
    coin = coin_for_symbol(symbol)

    if action == "position":
        address = wallet_address
        if not address:
            address = Account.from_key(private_key).address
        info = Info(api_url, skip_ws=True, spot_meta=EMPTY_SPOT_META)
        state = info.user_state(address)
        position = "0"
        for entry in state.get("assetPositions", []):
            pos = entry.get("position") or {}
            if pos.get("coin") == coin:
                position = str(pos.get("szi", "0"))
                break
        emit({"status": "ok", "position": position})
        return

    if action == "order":
        wallet = Account.from_key(private_key)
        exchange = Exchange(wallet, api_url, spot_meta=EMPTY_SPOT_META)
        leverage = int(payload.get("leverage", 5))
        is_cross = bool(payload.get("is_cross", True))
        reduce_only = bool(payload.get("reduce_only", False))
        exchange.update_leverage(leverage, coin, is_cross=is_cross)
        is_buy = str(payload["side"]).lower() == "long"
        size = float(payload["size"])
        if reduce_only:
            result = exchange.market_close(coin, size)
        else:
            result = exchange.market_open(coin, is_buy, size)
        oid, filled_size, filled_price = parse_filled(result)
        emit(
            {
                "status": "ok",
                "external_order_id": oid,
                "order_status": "filled",
                "filled_size": filled_size,
                "filled_price": filled_price,
                "raw": result,
            }
        )
        return

    emit({"status": "error", "error": f"unsupported action: {action}"})


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:  # noqa: BLE001
        emit({"status": "error", "error": str(exc)})
        raise SystemExit(1)
