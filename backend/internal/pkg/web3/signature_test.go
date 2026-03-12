package web3

import "testing"

func TestVerifyMessageSignature(t *testing.T) {
	const (
		address   = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
		message   = "Welcome to RGPerp!\n\nWallet: 0x70997970C51812dc3A010C7d01b50e0d17dc79C8\nNonce: demo-nonce\nDomain: localhost:18080\nChain ID: 31337\nTimestamp: 2026-03-12T00:00:00Z"
		signature = "0x392d9f2c3c3819846801d3f85be402b5ab2a0f8b61891a7e8c92041edf792a5d1c186ec66ef59d6340ce77358cf7841044f3d2ddf900b16ac90efea920f9a7441c"
	)

	if err := VerifyMessageSignature(message, signature, address); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
}
