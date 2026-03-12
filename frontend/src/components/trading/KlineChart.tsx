import { memo, useMemo } from 'react'
import { AdvancedChart } from 'react-tradingview-embed'

interface KlineChartProps {
  symbol: string
  interval: string
  dark?: boolean
  height?: number
}

function KlineChart({ symbol, interval, dark = true, height = 720 }: KlineChartProps) {
  const tradingViewSymbol = mapTradingViewSymbol(symbol)
  const tradingViewInterval = mapTradingViewInterval(interval)
  const tradingViewRange = mapTradingViewRange(interval)
  const containerId = useMemo(
    () => `rg-tv-${symbol.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-${interval}-${dark ? 'dark' : 'light'}`,
    [dark, interval, symbol],
  )
  const widgetProps = useMemo(
    () => ({
      symbol: tradingViewSymbol,
      interval: tradingViewInterval,
      range: tradingViewRange,
      theme: dark ? 'dark' : 'light',
      locale: 'en',
      timezone: 'Asia/Shanghai',
      width: '100%',
      height,
      autosize: false,
      allow_symbol_change: false,
      save_image: true,
      enable_publishing: false,
      withdateranges: true,
      hide_top_toolbar: false,
      hide_side_toolbar: false,
      style: '1',
      toolbar_bg: dark ? '#0b1720' : '#ffffff',
      container_id: containerId,
    }),
    [containerId, dark, height, tradingViewInterval, tradingViewRange, tradingViewSymbol],
  )

  return (
    <div className="rg-tv-chart-shell" style={{ height }}>
      <AdvancedChart widgetProps={widgetProps} />
    </div>
  )
}

export default memo(KlineChart)

function mapTradingViewSymbol(symbol: string) {
  switch (symbol) {
    case 'BTC-PERP':
      return 'BYBIT:BTCUSDT.P'
    case 'ETH-PERP':
      return 'BYBIT:ETHUSDT.P'
    case 'SOL-PERP':
      return 'BYBIT:SOLUSDT.P'
    default:
      return 'BYBIT:BTCUSDT.P'
  }
}

function mapTradingViewInterval(interval: string) {
  switch (interval) {
    case '1m':
      return '1'
    case '5m':
      return '5'
    case '15m':
      return '15'
    case '1h':
      return '60'
    case '1d':
      return '1D'
    default:
      return '1'
  }
}

function mapTradingViewRange(interval: string) {
  switch (interval) {
    case '1m':
      return '1D'
    case '5m':
      return '5D'
    case '15m':
      return '1M'
    case '1h':
      return '3M'
    case '1d':
      return '12M'
    default:
      return '1D'
  }
}
