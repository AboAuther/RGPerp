package service

import "strings"

var adminWallets = map[string]struct{}{}

func SetAdminWallets(wallets []string) {
	next := make(map[string]struct{}, len(wallets))
	for _, wallet := range wallets {
		normalized := strings.ToLower(strings.TrimSpace(wallet))
		if normalized == "" {
			continue
		}
		next[normalized] = struct{}{}
	}
	adminWallets = next
}

func IsAdminWallet(wallet string) bool {
	normalized := strings.ToLower(strings.TrimSpace(wallet))
	if normalized == "" {
		return false
	}
	_, ok := adminWallets[normalized]
	return ok
}
