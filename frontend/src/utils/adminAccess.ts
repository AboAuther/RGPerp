const defaultWallets = ['0x70997970c51812dc3a010c7d01b50e0d17dc79c8']

const configuredWallets = (import.meta.env.VITE_ADMIN_WALLETS ?? '')
  .split(',')
  .map((value: string) => value.trim().toLowerCase())
  .filter(Boolean)

export function isAdminWallet(walletAddress?: string | null) {
  if (!walletAddress) {
    return false
  }
  const allowlist = configuredWallets.length > 0 ? configuredWallets : defaultWallets
  return allowlist.includes(walletAddress.trim().toLowerCase())
}
