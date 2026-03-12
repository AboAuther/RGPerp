package web3

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func NormalizeAddress(address string) (string, error) {
	if !common.IsHexAddress(address) {
		return "", fmt.Errorf("invalid address")
	}
	return common.HexToAddress(address).Hex(), nil
}

func VerifyMessageSignature(message, signatureHex, expectedAddress string) error {
	address, err := NormalizeAddress(expectedAddress)
	if err != nil {
		return err
	}

	sig, err := hexutil.Decode(signatureHex)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(sig) != crypto.SignatureLength {
		return fmt.Errorf("invalid signature length")
	}

	if sig[64] >= 27 {
		sig[64] -= 27
	}
	if sig[64] > 1 {
		return fmt.Errorf("invalid recovery id")
	}

	hash := accounts.TextHash([]byte(message))
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return fmt.Errorf("recover signer: %w", err)
	}

	recovered := crypto.PubkeyToAddress(*pubKey).Hex()
	if recovered != address {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func SignWithdrawal(
	privateKeyHex string,
	userAddress string,
	amount uint64,
	nonce uint64,
	deadline uint64,
	chainID int64,
	vaultAddress string,
) (string, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	user := common.HexToAddress(userAddress)
	vault := common.HexToAddress(vaultAddress)
	hash := crypto.Keccak256Hash(
		user.Bytes(),
		newUint256(amount),
		newUint256(nonce),
		newUint256(deadline),
		newUint256(uint64(chainID)),
		vault.Bytes(),
	)
	sig, err := signPrefixedHash(privateKey, hash.Bytes())
	if err != nil {
		return "", err
	}
	return hexutil.Encode(sig), nil
}

func signPrefixedHash(privateKey *ecdsa.PrivateKey, messageHash []byte) ([]byte, error) {
	sig, err := crypto.Sign(accounts.TextHash(messageHash), privateKey)
	if err != nil {
		return nil, fmt.Errorf("sign hash: %w", err)
	}
	sig[64] += 27
	return sig, nil
}

func newUint256(v uint64) []byte {
	dst := make([]byte, 32)
	hexV := fmt.Sprintf("%064x", v)
	decoded, _ := hex.DecodeString(hexV)
	copy(dst, decoded)
	return dst
}
