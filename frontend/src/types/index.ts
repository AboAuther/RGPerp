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
  unrealized_pnl: string
  equity: string
  maintenance_margin: string
  margin_ratio: string
  risk_level: string
}

export interface Position {
  id: number
  symbol: string
  side: 'long' | 'short'
  margin_mode: 'isolated' | 'cross'
  size: string
  entry_price: string
  mark_price: string
  liquidation_price: string
  margin: string
  notional: string
  maintenance_margin: string
  risk_ratio: string
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
  margin_mode: 'isolated' | 'cross'
  size: string
  price: string
  leverage: number
  margin: string
  reduce_only: boolean
  limit_price?: string
  time_in_force?: string
  reserved_margin?: string
  reserved_fee?: string
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
  margin_mode: 'isolated' | 'cross'
  size: string
  limit_price: string
  time_in_force: string
  exec_price: string
  leverage: number
  margin: string
  reduce_only: boolean
  status: string
  fee: string
  realized_pnl: string
  created_at: string
  triggered_at?: string
  cancel_reason?: string
}

export interface TradeHistoryItem {
  id: number
  order_id: number
  symbol: string
  side: 'long' | 'short'
  size: string
  price: string
  fee: string
  realized_pnl: string
  is_liquidation: boolean
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
  change_24h: string
  change_24h_pct: string
  volume_24h: string
  open_interest: string
  funding_rate: string
  funding_next_at: number
  max_leverage: number
  display_pair: string
  base_asset: string
  quote_asset: string
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
  nonce: string
  signature: string
  deadline: string
  tx_hash?: string
  created_at: string
  rejection_reason?: string
}

export interface DepositRecord {
  tx_hash: string
  log_index: number
  block_number: number
  amount: string
  status: string
  created_at: string
}

export interface AdminOverview {
  trading_symbols: number
  open_positions: number
  accounts_at_risk: number
  accounts_reduce_only: number
  accounts_liquidating: number
  pending_hedges: number
  buffered_hedges: number
  retrying_hedges: number
  failed_hedges: number
  recent_liquidations: number
  unhealthy_symbols: number
  total_absolute_drift: string
  last_snapshot_at?: string
  drift_by_symbol?: Array<{
    symbol: string
    drift: string
    hedge_healthy: boolean
  }>
}

export interface AdminAlertItem {
  level: string
  category: string
  title: string
  detail: string
  symbol?: string
  created_at: string
}

export interface AdminHedgeTaskItem {
  id: number
  symbol: string
  trigger_type: string
  internal_net_position: string
  target_hedge_position: string
  current_hedge_position: string
  drift: string
  status: string
  error_message: string
  created_at: string
  updated_at: string
  last_order_status: string
  last_order_side: string
  last_order_size: string
  last_order_price: string
  last_order_retry_count: number
}

export interface AdminRiskSnapshotItem {
  id: number
  symbol: string
  total_long_position: string
  total_short_position: string
  net_position: string
  external_hedge_position: string
  drift: string
  hedge_healthy: boolean
  total_open_interest: string
  created_at: string
}

export interface AdminLiquidationItem {
  id: number
  user_id: number
  symbol: string
  side: string
  size: string
  entry_price: string
  mark_price: string
  liquidation_price: string
  execution_price: string
  realized_pnl: string
  type: string
  status: string
  created_at: string
}
