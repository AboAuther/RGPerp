// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Script.sol";
import "../src/MockUSDC.sol";
import "../src/Vault.sol";

contract DeployScript is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address operator = vm.envAddress("OPERATOR_ADDRESS");

        vm.startBroadcast(deployerPrivateKey);

        MockUSDC usdc = new MockUSDC();
        Vault vault = new Vault(address(usdc), operator);

        vm.stopBroadcast();

        console.log("MockUSDC deployed at:", address(usdc));
        console.log("Vault deployed at:", address(vault));
    }
}
