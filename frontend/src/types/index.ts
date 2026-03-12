export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data?: T
}

export interface PaginatedData<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface User {
  id: number
  wallet_address: string
  status: string
  created_at: string
}

export interface Account {
  asset: string
  available_balance: string
  locked_balance: string
  pending_withdrawal: string
  withdrawable_balance: string
  equity: string
}

export interface Position {
  id: number
  user_id: number
  symbol: string
  side: 'long' | 'short'
  size: string
  entry_price: string
  mark_price: string
  liquidation_price: string
  margin: string
  leverage: number
  unrealized_pnl: string
  realized_pnl: string
  status: string
}

export interface Order {
  id: number
  client_order_id: string
  user_id: number
  symbol: string
  side: 'long' | 'short'
  type: string
  size: string
  price: string
  leverage: number
  margin: string
  reduce_only: boolean
  status: string
  filled_size: string
  exec_price: string
  realized_pnl: string
  fee: string
  created_at: string
}

export interface OrderHistoryItem {
  id: number
  client_order_id: string
  symbol: string
  side: 'long' | 'short'
  type: string
  size: string
  exec_price: string
  leverage: number
  margin: string
  reduce_only: boolean
  status: string
  fee: string
  realized_pnl: string
  created_at: string
}

export interface OrderExecution {
  order: Order
  position?: Position
  account: {
    available_balance: string
    locked_balance: string
  }
}

export interface Trade {
  id: number
  order_id: number
  user_id: number
  symbol: string
  side: string
  size: string
  price: string
  fee: string
  realized_pnl: string
  is_liquidation: boolean
  created_at: string
}

export interface SymbolInfo {
  name: string
  base_asset: string
  quote_asset: string
  status: string
  max_leverage: number
  min_order_size: string
  tick_size: string
  lot_size: string
  taker_fee_rate: string
}

export interface PriceData {
  symbol: string
  mark_price: string
  index_price: string
  timestamp: number
}

export interface MarketTicker {
  symbol: string
  mark_price: string
  index_price: string
  best_bid: string
  best_ask: string
  source: string
  timestamp: number
}

export interface KlineItem {
  time: number
  open: string
  high: string
  low: string
  close: string
  volume: string
}

export interface AuthChallenge {
  nonce: string
  message: string
  expires_at: string
}

export interface AuthLogin {
  token: string
  expires_at: string
  user: {
    id: number
    wallet_address: string
    status: string
  }
}

export interface DepositInfo {
  chain_id: number
  vault_address: string
  usdc_address: string
  asset: string
  decimals: number
}

export interface WithdrawalRequest {
  request_id: string
  amount: string
  status: string
  nonce: number
  signature: string
  deadline: string
  tx_hash?: string
  created_at: string
}

export interface DepositRecord {
  tx_hash: string
  log_index: number
  block_number: number
  amount: string
  status: string
  created_at: string
}
