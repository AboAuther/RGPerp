# Cheatsheet

本文档提供了一些常用示例代码以及一些小提示，帮助你快速启动项目。

这些代码 **不是完整实现**，仅作为参考。

你可以自由选择是否使用。

---

# 1 钱包登录示例

常见 Web3 登录流程：

```
connect wallet
sign message
backend verify signature
```

前端示例（伪代码）：

```javascript
const message = "Login to Exchange"

const signature = await wallet.signMessage(message)

await fetch("/auth/login", {
  method: "POST",
  body: JSON.stringify({
    address: wallet.address,
    signature: signature
  })
})
```

后端验证示例：

```javascript
import { verifyMessage } from "ethers"

const recovered = verifyMessage(message, signature)

if (recovered !== address) {
  throw new Error("Invalid signature")
}
```

提示：这个message会不会有什么隐藏问题

---

# 2 Vault 合约模板（仅供参考）

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IERC20 {
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
    function transfer(address to, uint256 amount) external returns (bool);
}

contract Vault {

    IERC20 public usdc;

    mapping(address => uint256) public balances;

    event Deposit(address user, uint256 amount);
    event Withdraw(address user, uint256 amount);

    constructor(address _usdc) {
        usdc = IERC20(_usdc);
    }

    function deposit(uint256 amount) external {

        usdc.transferFrom(msg.sender, address(this), amount);

        balances[msg.sender] += amount;

        emit Deposit(msg.sender, amount);
    }

    function withdraw(uint256 amount) external {

        require(balances[msg.sender] >= amount, "insufficient");

        balances[msg.sender] -= amount;

        usdc.transfer(msg.sender, amount);

        emit Withdraw(msg.sender, amount);
    }
}
```

---

# 3 Event Indexer 示例

后端需要监听 Vault 合约事件。

示例（ethers.js）：

```javascript
vault.on("Deposit", (user, amount, event) => {

  console.log("deposit", user, amount)

  // update database
})
```
提示：是否需要考虑一些特殊问题，如幂等性？

---

# 4 PnL 计算

示例：

```
pnl = position_size * (mark_price - entry_price)
```

提示：

```
long 和 short 的计算方向不同，且entry_price和对冲机制会如何相互影响？
```

---

# 5 对冲示例

示例伪代码：

```javascript
async function hedge(symbol, size, side) {

  const order = {
    symbol,
    size,
    side
  }

  await hyperliquid.placeOrder(order)

}
```
提示：对冲是逐仓还是全仓？对系统设计有没有什么影响？如果在hyperliquid上的仓位被清算或者自动减仓了怎么办？

---

# 6 推荐项目结构

示例：

```
backend/

  src/
    auth/
    balance/
    order/
    position/
    engine/
    hedging/
    indexer/

frontend/

contracts/
```

---

# 7 推荐开发顺序

建议开发顺序：

```
1 wallet login
2 vault contract
3 deposit indexer
4 balance system
5 trading engine
6 position system
7 hedging system
8 frontend
```

---

# 8 建议优先完成的最小系统

如果时间不足，优先完成：

```
wallet login
deposit
trade
position
hedge
```

这是系统最小闭环。