package service

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

const oracleABI = `[
  {"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"getPrice","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"latestRoundData","outputs":[
    {"internalType":"uint80","name":"roundId","type":"uint80"},
    {"internalType":"int256","name":"answer","type":"int256"},
    {"internalType":"uint256","name":"startedAt","type":"uint256"},
    {"internalType":"uint256","name":"updatedAt","type":"uint256"},
    {"internalType":"uint80","name":"answeredInRound","type":"uint80"}
  ],"stateMutability":"view","type":"function"}
]`

type OracleReader struct {
	rpcURL string
}

func NewOracleReader(rpcURL string) *OracleReader {
	return &OracleReader{rpcURL: rpcURL}
}

func (r *OracleReader) GetPrice(ctx context.Context, feedAddress string) (decimal.Decimal, error) {
	client, err := ethclient.DialContext(ctx, r.rpcURL)
	if err != nil {
		return decimal.Zero, err
	}
	defer client.Close()

	parsedABI, err := abi.JSON(strings.NewReader(oracleABI))
	if err != nil {
		return decimal.Zero, err
	}
	contract := bind.NewBoundContract(common.HexToAddress(feedAddress), parsedABI, client, client, client)

	decimals := int32(8)
	{
		var out []interface{}
		if err := contract.Call(&bind.CallOpts{Context: ctx}, &out, "decimals"); err == nil && len(out) > 0 {
			if v, ok := out[0].(uint8); ok {
				decimals = int32(v)
			}
		}
	}

	// Try latestRoundData first (Chainlink-compatible).
	{
		var out []interface{}
		if err := contract.Call(&bind.CallOpts{Context: ctx}, &out, "latestRoundData"); err == nil && len(out) >= 2 {
			if answer, ok := out[1].(*big.Int); ok && answer != nil && answer.Sign() > 0 {
				return decimal.NewFromBigInt(answer, -decimals), nil
			}
		}
	}

	// Fallback to getPrice().
	{
		var out []interface{}
		if err := contract.Call(&bind.CallOpts{Context: ctx}, &out, "getPrice"); err != nil {
			return decimal.Zero, err
		}
		if len(out) > 0 {
			if price, ok := out[0].(*big.Int); ok && price != nil {
				return decimal.NewFromBigInt(price, -decimals), nil
			}
		}
	}
	return decimal.Zero, nil
}
