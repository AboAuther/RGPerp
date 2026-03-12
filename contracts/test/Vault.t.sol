// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/MockUSDC.sol";
import "../src/Vault.sol";

contract VaultTest is Test {
    MockUSDC public usdc;
    Vault public vault;

    address public owner = address(this);
    uint256 public operatorPrivateKey = 0xA11CE;
    address public operator = vm.addr(operatorPrivateKey);
    address public user = address(0xBEEF);

    function setUp() public {
        usdc = new MockUSDC();
        vault = new Vault(address(usdc), operator);

        usdc.mint(user, 100_000 * 1e6);
        vm.prank(user);
        usdc.approve(address(vault), type(uint256).max);
    }

    function test_Deposit() public {
        vm.prank(user);
        vault.deposit(1000 * 1e6);

        assertEq(usdc.balanceOf(address(vault)), 1000 * 1e6);
        assertEq(usdc.balanceOf(user), 99_000 * 1e6);
    }

    function test_Deposit_RevertZeroAmount() public {
        vm.prank(user);
        vm.expectRevert(Vault.InvalidAmount.selector);
        vault.deposit(0);
    }

    function test_Withdraw() public {
        vm.prank(user);
        vault.deposit(1000 * 1e6);

        uint256 amount = 500 * 1e6;
        uint256 nonce = 1;
        uint256 deadline = block.timestamp + 1 hours;

        bytes32 messageHash = keccak256(
            abi.encodePacked(user, amount, nonce, deadline, block.chainid, address(vault))
        );
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(operatorPrivateKey, ethSignedHash);
        bytes memory signature = abi.encodePacked(r, s, v);

        vm.prank(user);
        vault.withdraw(amount, nonce, deadline, signature);

        assertEq(usdc.balanceOf(user), 99_500 * 1e6);
        assertEq(usdc.balanceOf(address(vault)), 500 * 1e6);
        assertTrue(vault.usedNonces(nonce));
    }

    function test_Withdraw_RevertReplayNonce() public {
        vm.prank(user);
        vault.deposit(1000 * 1e6);

        uint256 amount = 100 * 1e6;
        uint256 nonce = 42;
        uint256 deadline = block.timestamp + 1 hours;

        bytes32 messageHash = keccak256(
            abi.encodePacked(user, amount, nonce, deadline, block.chainid, address(vault))
        );
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(operatorPrivateKey, ethSignedHash);
        bytes memory signature = abi.encodePacked(r, s, v);

        vm.prank(user);
        vault.withdraw(amount, nonce, deadline, signature);

        vm.prank(user);
        vm.expectRevert(Vault.NonceAlreadyUsed.selector);
        vault.withdraw(amount, nonce, deadline, signature);
    }

    function test_Withdraw_RevertExpiredDeadline() public {
        vm.prank(user);
        vault.deposit(1000 * 1e6);

        uint256 amount = 100 * 1e6;
        uint256 nonce = 1;
        uint256 deadline = block.timestamp - 1;

        bytes32 messageHash = keccak256(
            abi.encodePacked(user, amount, nonce, deadline, block.chainid, address(vault))
        );
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(operatorPrivateKey, ethSignedHash);
        bytes memory signature = abi.encodePacked(r, s, v);

        vm.prank(user);
        vm.expectRevert(Vault.DeadlineExpired.selector);
        vault.withdraw(amount, nonce, deadline, signature);
    }

    function test_SetOperator() public {
        address newOp = address(0xCAFE);
        vault.setOperator(newOp);
        assertEq(vault.operator(), newOp);
    }

    function test_SetOperator_RevertNotOwner() public {
        vm.prank(user);
        vm.expectRevert();
        vault.setOperator(address(0xCAFE));
    }
}
