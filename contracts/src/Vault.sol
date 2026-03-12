// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/// @title Vault - On-chain fund custody for PerpExchange
/// @notice Users deposit/withdraw USDC. Withdrawals require operator signature.
contract Vault is ReentrancyGuard, Ownable {
    using SafeERC20 for IERC20;

    IERC20 public immutable usdc;
    address public operator;
    mapping(uint256 => bool) public usedNonces;

    event Deposit(address indexed user, uint256 amount, uint256 timestamp);
    event Withdraw(address indexed user, uint256 amount, uint256 nonce, uint256 timestamp);
    event OperatorUpdated(address indexed oldOperator, address indexed newOperator);

    error InvalidAmount();
    error InvalidSignature();
    error NonceAlreadyUsed();
    error DeadlineExpired();
    error ZeroAddress();

    constructor(address _usdc, address _operator) Ownable(msg.sender) {
        if (_usdc == address(0) || _operator == address(0)) revert ZeroAddress();
        usdc = IERC20(_usdc);
        operator = _operator;
    }

    /// @notice Deposit USDC into the vault
    /// @param amount Amount of USDC to deposit (in USDC decimals)
    function deposit(uint256 amount) external nonReentrant {
        if (amount == 0) revert InvalidAmount();
        usdc.safeTransferFrom(msg.sender, address(this), amount);
        emit Deposit(msg.sender, amount, block.timestamp);
    }

    /// @notice Withdraw USDC with operator signature authorization
    /// @param amount Amount to withdraw
    /// @param nonce Unique nonce to prevent replay
    /// @param deadline Timestamp before which the signature is valid
    /// @param signature Operator's EIP-191 signature
    function withdraw(
        uint256 amount,
        uint256 nonce,
        uint256 deadline,
        bytes calldata signature
    ) external nonReentrant {
        if (amount == 0) revert InvalidAmount();
        if (block.timestamp > deadline) revert DeadlineExpired();
        if (usedNonces[nonce]) revert NonceAlreadyUsed();

        bytes32 messageHash = keccak256(
            abi.encodePacked(msg.sender, amount, nonce, deadline, block.chainid, address(this))
        );
        bytes32 ethSignedHash = _toEthSignedMessageHash(messageHash);

        if (_recoverSigner(ethSignedHash, signature) != operator) revert InvalidSignature();

        usedNonces[nonce] = true;
        usdc.safeTransfer(msg.sender, amount);
        emit Withdraw(msg.sender, amount, nonce, block.timestamp);
    }

    /// @notice Update operator address (owner only)
    function setOperator(address _operator) external onlyOwner {
        if (_operator == address(0)) revert ZeroAddress();
        emit OperatorUpdated(operator, _operator);
        operator = _operator;
    }

    function _toEthSignedMessageHash(bytes32 hash) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", hash));
    }

    function _recoverSigner(bytes32 hash, bytes calldata sig) internal pure returns (address) {
        if (sig.length != 65) revert InvalidSignature();
        bytes32 r;
        bytes32 s;
        uint8 v;
        assembly {
            r := calldataload(sig.offset)
            s := calldataload(add(sig.offset, 32))
            v := byte(0, calldataload(add(sig.offset, 64)))
        }
        if (v < 27) v += 27;
        return ecrecover(hash, v, r, s);
    }
}
