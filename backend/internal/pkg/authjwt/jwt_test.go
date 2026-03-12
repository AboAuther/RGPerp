package authjwt

import (
	"testing"
	"time"
)

func TestGenerateAndParse(t *testing.T) {
	token, err := Generate("test-secret", 42, "0xabc", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := Parse("test-secret", token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("unexpected user id: %d", claims.UserID)
	}
	if claims.WalletAddress != "0xabc" {
		t.Fatalf("unexpected wallet address: %s", claims.WalletAddress)
	}
}
