import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Form,
  Input,
  InputNumber,
  Row,
  Slider,
  Segmented,
  Select,
  Skeleton,
  Space,
  Switch,
  Table,
  Tag,
  Tabs,
  Typography,
  message,
} from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { get, post, getErrorMessage } from '../services/api'
import type {
  Account,
  AuthChallenge,
  AuthLogin,
  MarketTicker,
  OrderExecution,
  OrderHistoryItem,
  Position,
  SymbolInfo,
  TradeHistoryItem,
} from '../types'
import KlineChart from '../components/trading/KlineChart'
import { useThemeStore } from '../stores/themeStore'
import { useAuthStore } from '../stores/authStore'

function formatAmount(value?: string | number | null, precision = 6): string {
  if (value === null || value === undefined || value === '') {
    return '--'
  }
  const num = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(num)) {
    return String(value)
  }
  return num.toLocaleString(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: precision,
  })
}

function formatSignedAmount(value?: string | number | null, precision = 2): string {
  if (value === null || value === undefined || value === '') {
    return '--'
  }
  const num = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(num)) {
    return String(value)
  }
  const abs = Math.abs(num).toLocaleString(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: precision,
  })
  return `${num >= 0 ? '+' : '-'}${abs}`
}

function formatPairLabel(base?: string, quote?: string, fallback?: string): string {
  if (base && quote) {
    return `${base}/${quote}`
  }
  return fallback ?? '--'
}

function formatCountdown(timestamp?: number): string {
  if (!timestamp) {
    return '--:--:--'
  }
  const diff = Math.max(timestamp * 1000 - Date.now(), 0)
  const totalSeconds = Math.floor(diff / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return [hours, minutes, seconds].map((item) => String(item).padStart(2, '0')).join(':')
}

function riskMeta(level?: string) {
  switch ((level ?? '').toLowerCase()) {
    case 'active':
    case 'normal':
      return { label: '正常', color: 'green' as const, message: '账户风险状态正常，可正常交易。', type: 'success' as const }
    case 'at_risk':
      return { label: '风险上升', color: 'gold' as const, message: '账户已接近风控阈值，建议主动减仓或补充保证金。', type: 'warning' as const }
    case 'reduce_only':
      return { label: '只减仓', color: 'orange' as const, message: '账户已进入只减仓状态，暂时不能继续增加风险敞口。', type: 'warning' as const }
    case 'liquidating':
      return { label: '强平中', color: 'red' as const, message: '账户已进入强平流程，系统会自动减仓或全平风险仓位。', type: 'error' as const }
    case 'frozen':
      return { label: '冻结', color: 'magenta' as const, message: '账户已被冻结，请先排查风控或权限问题。', type: 'error' as const }
    default:
      return { label: level || '未知', color: 'default' as const, message: '当前风险状态未知，请刷新页面确认。', type: 'info' as const }
  }
}

function isTestLeveragePosition(leverage: number, maxLeverage?: number) {
  return leverage > (maxLeverage ?? 40)
}

function formatPositionRiskRatio(value?: string | number | null) {
  const ratio = Number(value ?? 0)
  if (!Number.isFinite(ratio)) {
    return {
      text: '--',
      color: '#8ca3b8',
      note: '当前仓位风险数据不可用。',
    }
  }
  if (ratio >= 100) {
    return {
      text: `${formatAmount(ratio, 2)}%`,
      color: '#ff6b6b',
      note: '仓位权益低于维持保证金要求，测试仓位会优先进入强平流程。',
    }
  }
  if (ratio >= 80) {
    return {
      text: `${formatAmount(ratio, 2)}%`,
      color: '#fbbf24',
      note: '已接近维持保证金阈值，建议尽快减仓或补充保证金。',
    }
  }
  return {
    text: `${formatAmount(ratio, 2)}%`,
    color: '#2ec9b0',
    note: '当前仓位权益仍覆盖维持保证金要求。',
  }
}

function getLiquidationBuffer(position?: Position) {
  if (!position) {
    return {
      title: '清算缓冲',
      valueText: '--',
      color: '#8ca3b8',
      note: '暂无持仓。',
    }
  }

  const markPrice = Number(position.mark_price)
  const liquidationPrice = Number(position.liquidation_price)
  if (!Number.isFinite(markPrice) || !Number.isFinite(liquidationPrice) || markPrice <= 0 || liquidationPrice <= 0) {
    return {
      title: '清算缓冲',
      valueText: 'N/A',
      color: '#8ca3b8',
      note: '当前仓位没有有效清算价，通常表示清算价已落到正常价格区间之外。',
    }
  }

  const rawPercent =
    position.side === 'long'
      ? ((markPrice - liquidationPrice) / markPrice) * 100
      : ((liquidationPrice - markPrice) / markPrice) * 100

  if (rawPercent < 0) {
    return {
      title: '已穿清算线',
      valueText: `${formatAmount(Math.abs(rawPercent), 2)}%`,
      color: '#ff6b6b',
      note: `当前标记价已经越过理论清算线，理论清算价 ${formatAmount(liquidationPrice, 4)}，liquidator 会优先处理这类仓位。`,
    }
  }

  if (rawPercent < 5) {
    return {
      title: '清算缓冲',
      valueText: `${formatAmount(rawPercent, 2)}%`,
      color: '#fbbf24',
      note: `理论清算价 ${formatAmount(liquidationPrice, 4)}，距离清算线非常近，价格轻微逆向波动就可能触发强平。`,
    }
  }

  return {
    title: '清算缓冲',
    valueText: `${formatAmount(rawPercent, 2)}%`,
    color: '#2ec9b0',
    note: `理论清算价 ${formatAmount(liquidationPrice, 4)}，当前价格距离清算线仍有一定空间。`,
  }
}

export default function TradePage() {
  const [symbol, setSymbol] = useState<string>('BTC-PERP')
  const [positionFilterSymbol, setPositionFilterSymbol] = useState<string>('BTC-PERP')
  const [openOrderFilterSymbol, setOpenOrderFilterSymbol] = useState<string>('BTC-PERP')
  const [tradeFilterSymbol, setTradeFilterSymbol] = useState<string>('BTC-PERP')
  const [orderHistoryFilterSymbol, setOrderHistoryFilterSymbol] = useState<string>('BTC-PERP')
  const [interval, setInterval] = useState<'1m' | '5m' | '15m' | '1h' | '1d'>('1m')
  const [orderForm] = Form.useForm()
  const [partialCloseSizes, setPartialCloseSizes] = useState<Record<number, string>>({})
  const { mode } = useThemeStore()
  const isDark = mode === 'dark'
  const queryClient = useQueryClient()
  const { isAuthenticated, setAuth } = useAuthStore()
  const authenticated = isAuthenticated()
  const [latestExecution, setLatestExecution] = useState<OrderExecution | null>(null)
  const [messageApi, contextHolder] = message.useMessage()

  const marketsQuery = useQuery({
    queryKey: ['markets'],
    queryFn: async () => (await get<SymbolInfo[]>('/markets')).data ?? [],
    refetchInterval: 30_000,
  })

  useEffect(() => {
    if (!marketsQuery.data?.length) {
      return
    }
    const hasSelected = marketsQuery.data.some((item) => item.name === symbol)
    if (!hasSelected) {
      setSymbol(marketsQuery.data[0].name)
    }
  }, [marketsQuery.data, symbol])

  useEffect(() => {
    setPositionFilterSymbol(symbol)
    setOpenOrderFilterSymbol(symbol)
    setTradeFilterSymbol(symbol)
    setOrderHistoryFilterSymbol(symbol)
  }, [symbol])

  const tickerQuery = useQuery({
    queryKey: ['ticker', symbol],
    queryFn: async () => (await get<MarketTicker>(`/markets/${symbol}/price`)).data!,
    enabled: Boolean(symbol),
    refetchInterval: 2_000,
  })

  const accountQuery = useQuery({
    queryKey: ['trade-account'],
    queryFn: async () => (await get<Account>('/account')).data!,
    enabled: authenticated,
  })

  const positionsQuery = useQuery({
    queryKey: ['positions'],
    queryFn: async () => (await get<Position[]>('/positions')).data ?? [],
    enabled: authenticated,
    refetchInterval: 4000,
  })

  const ordersQuery = useQuery({
    queryKey: ['orders'],
    queryFn: async () => (await get<OrderHistoryItem[]>('/orders', { limit: 20 })).data ?? [],
    enabled: authenticated,
    refetchInterval: 4000,
  })

  const openOrdersQuery = useQuery({
    queryKey: ['open-orders'],
    queryFn: async () => (await get<OrderHistoryItem[]>('/open-orders', { limit: 20 })).data ?? [],
    enabled: authenticated,
    refetchInterval: 4000,
  })

  const tradesQuery = useQuery({
    queryKey: ['trades'],
    queryFn: async () => (await get<TradeHistoryItem[]>('/trades', { limit: 20 })).data ?? [],
    enabled: authenticated,
    refetchInterval: 4000,
  })

  const loginMutation = useMutation({
    mutationFn: async () => {
      if (!window.ethereum) {
        throw new Error('wallet not found')
      }

      const accounts = (await window.ethereum.request({
        method: 'eth_requestAccounts',
      })) as string[]
      const selectedAddress = accounts?.[0]
      if (!selectedAddress) {
        throw new Error('wallet account not found')
      }

      const challenge = (
        await post<AuthChallenge>('/auth/challenge', {
          wallet_address: selectedAddress,
          domain: window.location.host,
          chain_id: Number(import.meta.env.VITE_CHAIN_ID || 31337),
        })
      ).data!

      const signature = (await window.ethereum.request({
        method: 'personal_sign',
        params: [challenge.message, selectedAddress],
      })) as string

      const login = (
        await post<AuthLogin>('/auth/login', {
          wallet_address: selectedAddress,
          nonce: challenge.nonce,
          signature,
        })
      ).data!

      return login
    },
    onSuccess: (result) => {
      setAuth(result.token, result.user.wallet_address)
      void queryClient.invalidateQueries({ queryKey: ['trade-account'] })
      void queryClient.invalidateQueries({ queryKey: ['positions'] })
      void queryClient.invalidateQueries({ queryKey: ['orders'] })
      void queryClient.invalidateQueries({ queryKey: ['open-orders'] })
      void queryClient.invalidateQueries({ queryKey: ['trades'] })
      void messageApi.success('登录成功')
    },
    onError: (error) => {
      void messageApi.error(getErrorMessage(error, '登录失败'))
    },
  })

  const orderMutation = useMutation({
    mutationFn: async (values: {
      side: 'long' | 'short'
      margin_mode: 'isolated' | 'cross'
      size: string
      leverage: number
      reduce_only: boolean
      test_mode: boolean
    }) =>
      (
        await post<OrderExecution>('/orders', {
          symbol,
          side: values.side,
          type: 'market',
          margin_mode: values.margin_mode,
          size: values.size,
          leverage: Number(values.leverage),
          reduce_only: values.reduce_only,
          test_mode: values.test_mode,
          client_order_id: `web-${Date.now()}`,
        })
      ).data!,
    onSuccess: (result) => {
      setLatestExecution(result)
      orderForm.resetFields(['size', 'reduce_only'])
      void queryClient.invalidateQueries({ queryKey: ['trade-account'] })
      void queryClient.invalidateQueries({ queryKey: ['positions'] })
      void queryClient.invalidateQueries({ queryKey: ['orders'] })
      void queryClient.invalidateQueries({ queryKey: ['open-orders'] })
      void queryClient.invalidateQueries({ queryKey: ['trades'] })
      void messageApi.success('市价单已成交')
    },
    onError: (error) => {
      void messageApi.error(getErrorMessage(error, '下单失败'))
    },
  })

  const submitOrder = (values: {
    side: 'long' | 'short'
    margin_mode: 'isolated' | 'cross'
    size: string
    leverage: number
    reduce_only: boolean
    test_mode: boolean
  }) => {
    orderMutation.mutate(values)
  }

  const submitPartialClose = (record: Position) => {
    const size = partialCloseSizes[record.id]
    if (!size || Number(size) <= 0) {
      void messageApi.error('请输入部分平仓数量')
      return
    }
    if (Number(size) > Number(record.size)) {
      void messageApi.error('部分平仓数量不能超过当前持仓')
      return
    }
    submitOrder({
      side: record.side === 'long' ? 'short' : 'long',
      margin_mode: record.margin_mode,
      size,
      leverage: record.leverage,
      reduce_only: true,
      test_mode: isTestLeveragePosition(record.leverage, currentSymbol?.max_leverage),
    })
  }

  const marketOptions = useMemo(
    () =>
      (marketsQuery.data ?? []).map((item) => ({
        label: formatPairLabel(item.base_asset, item.quote_asset, item.name),
        value: item.name,
      })),
    [marketsQuery.data],
  )

  const currentSymbol = (marketsQuery.data ?? []).find((item) => item.name === symbol)
  const pairFilterOptions = useMemo(
    () => [
      { label: '当前交易对', value: symbol },
      { label: '全部交易对', value: '__all__' },
      ...(marketsQuery.data ?? []).map((item) => ({
        label: formatPairLabel(item.base_asset, item.quote_asset, item.name),
        value: item.name,
      })),
    ],
    [marketsQuery.data, symbol],
  )
  const displayPair = formatPairLabel(currentSymbol?.base_asset, currentSymbol?.quote_asset, tickerQuery.data?.display_pair ?? 'BTC/USDC')
  const symbolLabelMap = useMemo(
    () =>
      new Map((marketsQuery.data ?? []).map((item) => [item.name, formatPairLabel(item.base_asset, item.quote_asset, item.name)])),
    [marketsQuery.data],
  )
  const markPrice = Number(tickerQuery.data?.mark_price ?? 0)
  const selectedLeverage = Number(Form.useWatch('leverage', orderForm) ?? 10)
  const selectedSize = Number(Form.useWatch('size', orderForm) ?? 0)
  const selectedSide = Form.useWatch('side', orderForm) ?? 'long'
  const selectedMarginMode = Form.useWatch('margin_mode', orderForm) ?? 'isolated'
  const testModeEnabled = Boolean(Form.useWatch('test_mode', orderForm))
  const leverageMax = testModeEnabled ? 1000 : currentSymbol?.max_leverage ?? 40
  const estimatedNotional = markPrice * selectedSize
  const estimatedMargin = selectedLeverage > 0 ? estimatedNotional / selectedLeverage : 0
  const takerFeeRate = Number(currentSymbol?.taker_fee_rate ?? 0)
  const estimatedFee = estimatedNotional * takerFeeRate

  useEffect(() => {
    if (selectedLeverage > leverageMax) {
      orderForm.setFieldValue('leverage', leverageMax)
    }
  }, [leverageMax, orderForm, selectedLeverage])
  const filteredPositions = useMemo(() => {
    if (positionFilterSymbol === '__all__') {
      return positionsQuery.data ?? []
    }
    return (positionsQuery.data ?? []).filter((item) => item.symbol === positionFilterSymbol)
  }, [positionFilterSymbol, positionsQuery.data])

  const filteredOpenOrders = useMemo(() => {
    if (openOrderFilterSymbol === '__all__') {
      return openOrdersQuery.data ?? []
    }
    return (openOrdersQuery.data ?? []).filter((item) => item.symbol === openOrderFilterSymbol)
  }, [openOrderFilterSymbol, openOrdersQuery.data])

  const filteredTrades = useMemo(() => {
    if (tradeFilterSymbol === '__all__') {
      return tradesQuery.data ?? []
    }
    return (tradesQuery.data ?? []).filter((item) => item.symbol === tradeFilterSymbol)
  }, [tradeFilterSymbol, tradesQuery.data])

  const filteredOrderHistory = useMemo(() => {
    if (orderHistoryFilterSymbol === '__all__') {
      return ordersQuery.data ?? []
    }
    return (ordersQuery.data ?? []).filter((item) => item.symbol === orderHistoryFilterSymbol)
  }, [orderHistoryFilterSymbol, ordersQuery.data])
  const accountRisk = riskMeta(accountQuery.data?.risk_level)
  const latestLiquidationTrade = useMemo(
    () =>
      (tradesQuery.data ?? [])
        .filter((item) => item.is_liquidation && item.symbol === symbol)
        .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())[0],
    [symbol, tradesQuery.data],
  )

  const positionColumns = useMemo(
    () => [
      { title: '交易对', dataIndex: 'symbol', key: 'symbol', width: 110, render: (value: string) => symbolLabelMap.get(value) ?? value },
      {
        title: '方向',
        dataIndex: 'side',
        key: 'side',
        width: 82,
        render: (value: string) => <Tag color={value === 'long' ? 'green' : 'red'}>{value}</Tag>,
      },
      {
        title: '模式',
        dataIndex: 'margin_mode',
        key: 'margin_mode',
        width: 90,
        render: (value: string) => <Tag color={value === 'cross' ? 'purple' : 'gold'}>{value}</Tag>,
      },
      { title: '数量', dataIndex: 'size', key: 'size', width: 84, render: (value: string) => formatAmount(value, 4) },
      { title: '开仓价', dataIndex: 'entry_price', key: 'entry_price', width: 94, render: (value: string) => formatAmount(value, 2) },
      { title: '标记价', dataIndex: 'mark_price', key: 'mark_price', width: 94, render: (value: string) => formatAmount(value, 2) },
      { title: '保证金', dataIndex: 'margin', key: 'margin', width: 112, render: (value: string) => `${formatAmount(value)} USDC` },
      {
        title: '未实现盈亏',
        dataIndex: 'unrealized_pnl',
        key: 'unrealized_pnl',
        width: 110,
        render: (value: string) => (
          <Typography.Text style={{ color: Number(value) >= 0 ? '#2ec9b0' : '#ff6b6b' }}>{formatAmount(value)}</Typography.Text>
        ),
      },
      {
        title: '维持保证金占权益比',
        dataIndex: 'risk_ratio',
        key: 'risk_ratio',
        width: 128,
        render: (value: string) => {
          const risk = formatPositionRiskRatio(value)
          return <Typography.Text style={{ color: risk.color }}>{risk.text}</Typography.Text>
        },
      },
      { title: '清算价', dataIndex: 'liquidation_price', key: 'liquidation_price', width: 84, render: (value: string) => formatAmount(value, 2) },
      {
        title: '操作',
        key: 'actions',
        width: 312,
        render: (_: unknown, record: Position) => (
          <div className="rg-position-actions">
            <InputNumber
              className="rg-position-actions-input"
              size="small"
              min={0}
              max={Number(record.size)}
              step={Number(record.size) >= 1 ? 0.1 : 0.001}
              placeholder="部分平仓"
              value={partialCloseSizes[record.id] ? Number(partialCloseSizes[record.id]) : undefined}
              onChange={(value) =>
                setPartialCloseSizes((current) => ({
                  ...current,
                  [record.id]: value === null ? '' : String(value),
                }))
              }
            />
            <Button
              size="small"
              className="rg-action-button"
              onClick={() => submitPartialClose(record)}
              loading={orderMutation.isPending}
            >
              部分平仓
            </Button>
            <Button
              size="small"
              className="rg-action-button"
              onClick={() =>
                submitOrder({
                  side: record.side === 'long' ? 'short' : 'long',
                  margin_mode: record.margin_mode,
                  size: record.size,
                  leverage: record.leverage,
                  reduce_only: true,
                  test_mode: isTestLeveragePosition(record.leverage, currentSymbol?.max_leverage),
                })
              }
              loading={orderMutation.isPending}
            >
              平仓
            </Button>
            <Button
              size="small"
              type="primary"
              ghost
              className="rg-action-button"
              onClick={() =>
                submitOrder({
                  side: record.side === 'long' ? 'short' : 'long',
                  margin_mode: record.margin_mode,
                  size: String(Number(record.size) * 2),
                  leverage: record.leverage,
                  reduce_only: false,
                  test_mode: isTestLeveragePosition(record.leverage, currentSymbol?.max_leverage),
                })
              }
              loading={orderMutation.isPending}
            >
              反手
            </Button>
          </div>
        ),
      },
    ],
    [currentSymbol?.max_leverage, messageApi, orderMutation.isPending, partialCloseSizes, symbolLabelMap],
  )

  const tradeColumns = useMemo(
    () => [
      { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 168, render: (value: string) => new Date(value).toLocaleString() },
      { title: '交易对', dataIndex: 'symbol', key: 'symbol', width: 110, render: (value: string) => symbolLabelMap.get(value) ?? value },
      { title: '方向', dataIndex: 'side', key: 'side', width: 82, render: (value: string) => <Tag color={value === 'long' ? 'green' : 'red'}>{value}</Tag> },
      { title: '数量', dataIndex: 'size', key: 'size', width: 84, render: (value: string) => formatAmount(value, 4) },
      { title: '成交价', dataIndex: 'price', key: 'price', width: 96, render: (value: string) => formatAmount(value, 2) },
      { title: '手续费', dataIndex: 'fee', key: 'fee', width: 94, render: (value: string) => formatAmount(value) },
      {
        title: '已实现盈亏',
        dataIndex: 'realized_pnl',
        key: 'realized_pnl',
        width: 110,
        render: (value: string) => (
          <Typography.Text style={{ color: Number(value) >= 0 ? '#2ec9b0' : '#ff6b6b' }}>{formatAmount(value)}</Typography.Text>
        ),
      },
      {
        title: '类型',
        dataIndex: 'is_liquidation',
        key: 'is_liquidation',
        width: 120,
        render: (value: boolean) => <Tag color={value ? 'volcano' : 'geekblue'}>{value ? 'liquidation' : 'trade'}</Tag>,
      },
    ],
    [symbolLabelMap],
  )

  const topPosition = (positionsQuery.data ?? [])[0]
  const positionRisk = formatPositionRiskRatio(topPosition?.risk_ratio)
  const liquidationBuffer = getLiquidationBuffer(topPosition)

  const orderColumns = useMemo(
    () => [
      { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 168, render: (value: string) => new Date(value).toLocaleString() },
      { title: '交易对', dataIndex: 'symbol', key: 'symbol', width: 110, render: (value: string) => symbolLabelMap.get(value) ?? value },
      {
        title: '方向',
        dataIndex: 'side',
        key: 'side',
        width: 82,
        render: (value: string) => <Tag color={value === 'long' ? 'green' : 'red'}>{value}</Tag>,
      },
      { title: '模式', dataIndex: 'margin_mode', key: 'margin_mode', width: 92, render: (value: string) => <Tag color={value === 'cross' ? 'purple' : 'gold'}>{value}</Tag> },
      { title: '数量', dataIndex: 'size', key: 'size', width: 84, render: (value: string) => formatAmount(value, 4) },
      { title: '成交价', dataIndex: 'exec_price', key: 'exec_price', width: 96, render: (value: string) => formatAmount(value, 2) },
      { title: '杠杆', dataIndex: 'leverage', key: 'leverage', width: 72, render: (value: number) => `${value}x` },
      { title: '手续费', dataIndex: 'fee', key: 'fee', width: 94, render: (value: string) => formatAmount(value) },
      {
        title: '已实现盈亏',
        dataIndex: 'realized_pnl',
        key: 'realized_pnl',
        width: 110,
        render: (value: string) => (
          <Typography.Text style={{ color: Number(value) >= 0 ? '#2ec9b0' : '#ff6b6b' }}>{formatAmount(value)}</Typography.Text>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 92,
        render: (value: string) => <Tag color={value === 'filled' ? 'cyan' : 'default'}>{value}</Tag>,
      },
    ],
    [symbolLabelMap],
  )

  return (
    <div className="rg-app-page rg-app-page--trade">
      {contextHolder}
      <Card
        style={{ marginBottom: 16, border: 'none' }}
        styles={{ body: { padding: 18 } }}
        className="rg-glass-card"
      >
        <Row gutter={[20, 16]} align="middle" wrap>
          <Col flex="260px">
            <Space size={12} align="center">
              <Select
                value={symbol}
                options={marketOptions}
                onChange={(v) => setSymbol(v)}
                style={{ minWidth: 220 }}
                loading={marketsQuery.isLoading}
                placeholder="选择交易对"
                size="large"
              />
              <Tag color="cyan" style={{ marginInlineEnd: 0, fontSize: 14, paddingInline: 10, lineHeight: '28px' }}>
                {currentSymbol?.max_leverage ?? tickerQuery.data?.max_leverage ?? 40}x
              </Tag>
            </Space>
          </Col>
          <Col flex="auto">
            <Row gutter={[20, 12]}>
              <Col xs={12} md={8} xl={4}>
                <Typography.Text type="secondary">Mark</Typography.Text>
                <Typography.Title level={4} style={{ margin: 0, color: '#2ec9b0' }}>
                  {formatAmount(tickerQuery.data?.mark_price, 2)}
                </Typography.Title>
              </Col>
              <Col xs={12} md={8} xl={4}>
                <Typography.Text type="secondary">Oracle / Index</Typography.Text>
                <Typography.Title level={4} style={{ margin: 0 }}>
                  {formatAmount(tickerQuery.data?.index_price, 2)}
                </Typography.Title>
              </Col>
              <Col xs={12} md={8} xl={4}>
                <Typography.Text type="secondary">24h Change</Typography.Text>
                <Typography.Title
                  level={4}
                  style={{ margin: 0, color: Number(tickerQuery.data?.change_24h ?? 0) >= 0 ? '#2ec9b0' : '#ff6b6b' }}
                >
                  {`${formatSignedAmount(tickerQuery.data?.change_24h, 2)} / ${formatSignedAmount(tickerQuery.data?.change_24h_pct, 2)}%`}
                </Typography.Title>
              </Col>
              <Col xs={12} md={8} xl={4}>
                <Typography.Text type="secondary">24h Volume</Typography.Text>
                <Typography.Title level={4} style={{ margin: 0 }}>
                  ${formatAmount(tickerQuery.data?.volume_24h, 2)}
                </Typography.Title>
              </Col>
              <Col xs={12} md={8} xl={4}>
                <Typography.Text type="secondary">Open Interest</Typography.Text>
                <Typography.Title level={4} style={{ margin: 0 }}>
                  ${formatAmount(tickerQuery.data?.open_interest, 2)}
                </Typography.Title>
              </Col>
              <Col xs={12} md={8} xl={4}>
                <Typography.Text type="secondary">Funding / Countdown</Typography.Text>
                <Typography.Title
                  level={4}
                  style={{ margin: 0, color: Number(tickerQuery.data?.funding_rate ?? 0) >= 0 ? '#2ec9b0' : '#ff6b6b' }}
                >
                  {`${formatSignedAmount(Number(tickerQuery.data?.funding_rate ?? 0) * 100, 4)}%  ${formatCountdown(
                    tickerQuery.data?.funding_next_at,
                  )}`}
                </Typography.Title>
              </Col>
            </Row>
          </Col>
        </Row>
      </Card>
      <Row gutter={[16, 16]} align="stretch">
        <Col xs={24} lg={17} style={{ display: 'flex' }}>
          <Card
            title="K 线图表"
            className="rg-glass-card"
            style={{ width: '100%' }}
            styles={{ body: { display: 'flex', flexDirection: 'column', minHeight: 900 } }}
          >
            <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
              <Col>
                <Space size={10}>
                  <Typography.Text strong>{displayPair}</Typography.Text>
                  <Tag color="cyan">TradingView</Tag>
                  <Tag color="blue">Binance Futures</Tag>
                </Space>
              </Col>
            </Row>
            <div style={{ flex: 1, minHeight: 760 }}>
              <KlineChart symbol={currentSymbol?.name ?? symbol} interval={interval} dark={isDark} height={760} />
            </div>
            <Space size={16} style={{ marginTop: 16 }} wrap>
              <Typography.Text type="secondary">Best Bid {formatAmount(tickerQuery.data?.best_bid, 2)}</Typography.Text>
              <Typography.Text type="secondary">Best Ask {formatAmount(tickerQuery.data?.best_ask, 2)}</Typography.Text>
              <Typography.Text type="secondary">
                Updated {tickerQuery.data ? new Date(tickerQuery.data.timestamp * 1000).toLocaleTimeString() : '--'}
              </Typography.Text>
            </Space>
          </Card>
        </Col>
        <Col xs={24} lg={7} style={{ display: 'flex' }}>
          <Card
            title="下单面板"
            className="rg-glass-card"
            style={{ width: '100%' }}
            styles={{ body: { display: 'flex', flexDirection: 'column', minHeight: 900 } }}
          >
            {!authenticated ? (
              <Alert
                type="info"
                showIcon
                message="连接钱包后下单"
                description={
                  <Button type="primary" style={{ marginTop: 12 }} loading={loginMutation.isPending} onClick={() => loginMutation.mutate()}>
                    钱包登录
                  </Button>
                }
              />
            ) : (
              <Space direction="vertical" size={16} style={{ width: '100%' }}>
                <Alert
                  type={accountRisk.type}
                  showIcon
                  message={`账户风险状态：${accountRisk.label}`}
                  description={accountRisk.message}
                />
                <Card size="small" className="rg-glass-card">
                  <Row gutter={[12, 12]}>
                    <Col span={12}>
                      <Typography.Text type="secondary">可用余额</Typography.Text>
                      <Typography.Title level={4} style={{ margin: 0 }}>
                        {formatAmount(accountQuery.data?.available_balance)}
                      </Typography.Title>
                    </Col>
                    <Col span={12}>
                      <Typography.Text type="secondary">锁定保证金</Typography.Text>
                      <Typography.Title level={4} style={{ margin: 0 }}>
                        {formatAmount(accountQuery.data?.locked_balance)}
                      </Typography.Title>
                    </Col>
                  </Row>
                </Card>
                <Form
                  form={orderForm}
                  layout="vertical"
                  initialValues={{ side: 'long', margin_mode: 'isolated', leverage: 10, reduce_only: false, test_mode: false }}
                  onFinish={(values) => orderMutation.mutate(values)}
                >
                  <Form.Item label="方向" name="side">
                    <Segmented
                      block
                      className={`rg-side-segmented rg-side-segmented--${selectedSide}`}
                      options={[{ label: '做多', value: 'long' }, { label: '做空', value: 'short' }]}
                    />
                  </Form.Item>
                  <Form.Item label="保证金模式" name="margin_mode">
                    <Segmented block options={[{ label: '逐仓', value: 'isolated' }, { label: '全仓', value: 'cross' }]} />
                  </Form.Item>
                  <Form.Item label="测试专用" name="test_mode" valuePropName="checked">
                    <Switch checkedChildren="1000x 测试" unCheckedChildren="常规" />
                  </Form.Item>
                  {testModeEnabled ? (
                    <Alert
                      type="warning"
                      showIcon
                      style={{ marginBottom: 16 }}
                      message="测试专用高杠杆"
                      description="已开启测试模式。当前交易对可临时使用最高 1000x 杠杆，仅用于自动清算与强平联调。"
                    />
                  ) : null}
                  <Form.Item
                    label="数量"
                    name="size"
                    rules={[{ required: true, message: '请输入下单数量' }]}
                  >
                    <Input placeholder="例如 0.001" />
                  </Form.Item>
                  <Form.Item
                    label="杠杆"
                    name="leverage"
                    rules={[{ required: true, message: '请输入杠杆' }]}
                  >
                    <InputNumber min={1} max={leverageMax} style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item label={`杠杆滑块 ${selectedLeverage}x`}>
                    <Slider min={1} max={leverageMax} value={selectedLeverage} onChange={(value) => orderForm.setFieldValue('leverage', value)} />
                  </Form.Item>
                  <Form.Item label="只减仓" name="reduce_only" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Card size="small" style={{ marginBottom: 16 }}>
                    <Descriptions size="small" column={1}>
                      <Descriptions.Item label="方向">{selectedSide === 'long' ? '做多 Long' : '做空 Short'}</Descriptions.Item>
                      <Descriptions.Item label="模式">{selectedMarginMode === 'cross' ? '全仓 Cross' : '逐仓 Isolated'}</Descriptions.Item>
                      <Descriptions.Item label="名义价值">{formatAmount(estimatedNotional, 2)} USDC</Descriptions.Item>
                      <Descriptions.Item label="预估开仓保证金">{formatAmount(estimatedMargin)} USDC</Descriptions.Item>
                      <Descriptions.Item label="预估手续费">{formatAmount(estimatedFee)} USDC ({formatAmount(takerFeeRate * 100, 4)}%)</Descriptions.Item>
                    </Descriptions>
                  </Card>
                  <Button type="primary" htmlType="submit" block loading={orderMutation.isPending}>
                    市价下单
                  </Button>
                </Form>

                {latestExecution ? (
                  <Card size="small">
                    <Descriptions size="small" column={1}>
                      <Descriptions.Item label="成交价">{formatAmount(latestExecution.order.exec_price, 2)}</Descriptions.Item>
                      <Descriptions.Item label="保证金模式">{latestExecution.order.margin_mode === 'cross' ? '全仓 Cross' : '逐仓 Isolated'}</Descriptions.Item>
                      <Descriptions.Item label="手续费">{formatAmount(latestExecution.order.fee)}</Descriptions.Item>
                      <Descriptions.Item label="可用余额">{formatAmount(latestExecution.account.available_balance)}</Descriptions.Item>
                    </Descriptions>
                  </Card>
                ) : null}
              </Space>
            )}
          </Card>
        </Col>
        <Col xs={24}>
          <Card className="rg-glass-card">
            {authenticated ? (
              <Space direction="vertical" size={12} style={{ width: '100%', marginBottom: 16 }}>
                <Alert
                  type={accountRisk.type}
                  showIcon
                  message={`当前账户状态：${accountRisk.label}`}
                  description={`风险率 ${formatAmount(accountQuery.data?.margin_ratio, 2)}%，维持保证金 ${formatAmount(
                    accountQuery.data?.maintenance_margin,
                  )} USDC。`}
                />
                {latestLiquidationTrade ? (
                  <Alert
                    type="error"
                    showIcon
                    message="检测到最近强平"
                    description={`${symbolLabelMap.get(latestLiquidationTrade.symbol) ?? latestLiquidationTrade.symbol} 在 ${new Date(
                      latestLiquidationTrade.created_at,
                    ).toLocaleString()} 被系统强平，成交价 ${formatAmount(latestLiquidationTrade.price, 2)}，实现盈亏 ${formatAmount(
                      latestLiquidationTrade.realized_pnl,
                    )} USDC。`}
                  />
                ) : null}
              </Space>
            ) : null}
            {authenticated && topPosition ? (
              <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
                <Col xs={24} md={8} style={{ display: 'flex' }}>
                  <Card size="small" className="rg-glass-card rg-risk-summary-card" style={{ width: '100%' }}>
                    <Typography.Text type="secondary">未实现盈亏</Typography.Text>
                    <Typography.Title level={4} style={{ margin: 0, color: Number(topPosition.unrealized_pnl) >= 0 ? '#2ec9b0' : '#ff6b6b' }}>
                      {formatAmount(topPosition.unrealized_pnl)} USDC
                    </Typography.Title>
                  </Card>
                </Col>
                <Col xs={24} md={8} style={{ display: 'flex' }}>
                  <Card size="small" className="rg-glass-card rg-risk-summary-card" style={{ width: '100%' }}>
                    <Typography.Text type="secondary">维持保证金占权益比</Typography.Text>
                    <Typography.Title level={4} style={{ margin: 0, color: positionRisk.color }}>
                      {positionRisk.text}
                    </Typography.Title>
                    <Typography.Text type="secondary">{positionRisk.note}</Typography.Text>
                  </Card>
                </Col>
                <Col xs={24} md={8} style={{ display: 'flex' }}>
                  <Card size="small" className="rg-glass-card rg-risk-summary-card" style={{ width: '100%' }}>
                    <Typography.Text type="secondary">{liquidationBuffer.title}</Typography.Text>
                    <Typography.Title level={4} style={{ margin: 0, color: liquidationBuffer.color }}>
                      {liquidationBuffer.valueText}
                    </Typography.Title>
                    <Typography.Text type="secondary">{liquidationBuffer.note}</Typography.Text>
                  </Card>
                </Col>
              </Row>
            ) : null}

            <Tabs
              items={[
                {
                  key: 'positions',
                  label: 'Positions',
                  children: !authenticated ? (
                    <Typography.Text type="secondary">登录后可查看实时持仓与盈亏。</Typography.Text>
                  ) : (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <div className="rg-positions-toolbar">
                        <Typography.Text type="secondary">仓位筛选</Typography.Text>
                        <Select
                          value={positionFilterSymbol}
                          options={pairFilterOptions}
                          onChange={(value) => setPositionFilterSymbol(value)}
                          style={{ minWidth: 180 }}
                          size="small"
                        />
                      </div>
                      <Table
                        rowKey="id"
                        loading={positionsQuery.isLoading}
                        dataSource={filteredPositions}
                        columns={positionColumns}
                        pagination={false}
                        locale={{ emptyText: '当前筛选条件下无持仓' }}
                        scroll={{ x: 1260 }}
                      />
                    </Space>
                  ),
                },
                {
                  key: 'open-orders',
                  label: 'Open Orders',
                  children: !authenticated ? (
                    <Typography.Text type="secondary">登录后可查看挂单。</Typography.Text>
                  ) : (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <div className="rg-positions-toolbar">
                        <Typography.Text type="secondary">挂单筛选</Typography.Text>
                        <Select
                          value={openOrderFilterSymbol}
                          options={pairFilterOptions}
                          onChange={(value) => setOpenOrderFilterSymbol(value)}
                          style={{ minWidth: 180 }}
                          size="small"
                        />
                      </div>
                      <Table
                        rowKey="id"
                        loading={openOrdersQuery.isLoading}
                        dataSource={filteredOpenOrders}
                        columns={orderColumns}
                        pagination={false}
                        locale={{ emptyText: '当前筛选条件下无挂单' }}
                        scroll={{ x: 1040 }}
                      />
                    </Space>
                  ),
                },
                {
                  key: 'trade-history',
                  label: 'Trade History',
                  children: !authenticated ? (
                    <Typography.Text type="secondary">登录后可查看成交记录。</Typography.Text>
                  ) : (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <div className="rg-positions-toolbar">
                        <Typography.Text type="secondary">成交筛选</Typography.Text>
                        <Select
                          value={tradeFilterSymbol}
                          options={pairFilterOptions}
                          onChange={(value) => setTradeFilterSymbol(value)}
                          style={{ minWidth: 180 }}
                          size="small"
                        />
                      </div>
                      <Table
                        rowKey="id"
                        loading={tradesQuery.isLoading}
                        dataSource={filteredTrades}
                        columns={tradeColumns}
                        pagination={false}
                        locale={{ emptyText: '当前筛选条件下无成交记录' }}
                        scroll={{ x: 980 }}
                      />
                    </Space>
                  ),
                },
                {
                  key: 'order-history',
                  label: 'Order History',
                  children: !authenticated ? (
                    <Typography.Text type="secondary">登录后可查看订单历史。</Typography.Text>
                  ) : (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <div className="rg-positions-toolbar">
                        <Typography.Text type="secondary">订单筛选</Typography.Text>
                        <Select
                          value={orderHistoryFilterSymbol}
                          options={pairFilterOptions}
                          onChange={(value) => setOrderHistoryFilterSymbol(value)}
                          style={{ minWidth: 180 }}
                          size="small"
                        />
                      </div>
                      <Table
                        rowKey="id"
                        loading={ordersQuery.isLoading}
                        dataSource={filteredOrderHistory}
                        columns={orderColumns}
                        pagination={false}
                        locale={{ emptyText: '当前筛选条件下无订单记录' }}
                        scroll={{ x: 1040 }}
                      />
                    </Space>
                  ),
                },
              ]}
            />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
