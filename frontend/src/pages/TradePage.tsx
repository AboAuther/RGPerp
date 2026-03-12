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
import { get, post } from '../services/api'
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

export default function TradePage() {
  const [symbol, setSymbol] = useState<string>('BTC-PERP')
  const [interval, setInterval] = useState<'1m' | '5m' | '15m' | '1h' | '1d'>('1m')
  const [orderForm] = Form.useForm()
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
      void messageApi.error(error instanceof Error ? error.message : '登录失败')
    },
  })

  const orderMutation = useMutation({
    mutationFn: async (values: {
      side: 'long' | 'short'
      size: string
      leverage: number
      reduce_only: boolean
    }) =>
      (
        await post<OrderExecution>('/orders', {
          symbol,
          side: values.side,
          type: 'market',
          size: values.size,
          leverage: Number(values.leverage),
          reduce_only: values.reduce_only,
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
      void messageApi.error(error instanceof Error ? error.message : '下单失败')
    },
  })

  const marketOptions = useMemo(
    () =>
      (marketsQuery.data ?? []).map((item) => ({
        label: formatPairLabel(item.base_asset, item.quote_asset, item.name),
        value: item.name,
      })),
    [marketsQuery.data],
  )

  const currentSymbol = (marketsQuery.data ?? []).find((item) => item.name === symbol)
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
  const estimatedNotional = markPrice * selectedSize
  const estimatedMargin = selectedLeverage > 0 ? estimatedNotional / selectedLeverage : 0
  const estimatedFee = currentSymbol ? estimatedNotional * Number(currentSymbol.taker_fee_rate ?? 0) : 0

  const positionColumns = useMemo(
    () => [
      { title: '交易对', dataIndex: 'symbol', key: 'symbol', render: (value: string) => symbolLabelMap.get(value) ?? value },
      { title: '方向', dataIndex: 'side', key: 'side' },
      { title: '数量', dataIndex: 'size', key: 'size', render: (value: string) => formatAmount(value, 4) },
      { title: '开仓价', dataIndex: 'entry_price', key: 'entry_price', render: (value: string) => formatAmount(value, 2) },
      { title: '标记价', dataIndex: 'mark_price', key: 'mark_price', render: (value: string) => formatAmount(value, 2) },
      { title: '保证金', dataIndex: 'margin', key: 'margin', render: (value: string) => `${formatAmount(value)} USDC` },
      {
        title: '未实现盈亏',
        dataIndex: 'unrealized_pnl',
        key: 'unrealized_pnl',
        render: (value: string) => (
          <Typography.Text style={{ color: Number(value) >= 0 ? '#2ec9b0' : '#ff6b6b' }}>{formatAmount(value)}</Typography.Text>
        ),
      },
      { title: '清算价', dataIndex: 'liquidation_price', key: 'liquidation_price', render: (value: string) => formatAmount(value, 2) },
    ],
    [symbolLabelMap],
  )

  const tradeColumns = useMemo(
    () => [
      { title: '时间', dataIndex: 'created_at', key: 'created_at', render: (value: string) => new Date(value).toLocaleString() },
      { title: '交易对', dataIndex: 'symbol', key: 'symbol', render: (value: string) => symbolLabelMap.get(value) ?? value },
      { title: '方向', dataIndex: 'side', key: 'side', render: (value: string) => <Tag color={value === 'long' ? 'green' : 'red'}>{value}</Tag> },
      { title: '数量', dataIndex: 'size', key: 'size', render: (value: string) => formatAmount(value, 4) },
      { title: '成交价', dataIndex: 'price', key: 'price', render: (value: string) => formatAmount(value, 2) },
      { title: '手续费', dataIndex: 'fee', key: 'fee', render: (value: string) => formatAmount(value) },
      {
        title: '已实现盈亏',
        dataIndex: 'realized_pnl',
        key: 'realized_pnl',
        render: (value: string) => (
          <Typography.Text style={{ color: Number(value) >= 0 ? '#2ec9b0' : '#ff6b6b' }}>{formatAmount(value)}</Typography.Text>
        ),
      },
      { title: '类型', dataIndex: 'is_liquidation', key: 'is_liquidation', render: (value: boolean) => <Tag color={value ? 'volcano' : 'geekblue'}>{value ? 'liquidation' : 'trade'}</Tag> },
    ],
    [symbolLabelMap],
  )

  const topPosition = (positionsQuery.data ?? [])[0]
  const maintenanceMargin = topPosition ? Number(topPosition.margin) * 0.5 : 0
  const marginRatio = topPosition && Number(topPosition.margin) > 0 ? (maintenanceMargin / Number(topPosition.margin)) * 100 : 0
  const liquidationDistance = topPosition
    ? ((Number(topPosition.mark_price) - Number(topPosition.liquidation_price)) / Math.max(Number(topPosition.mark_price), 1)) * 100
    : 0

  const orderColumns = useMemo(
    () => [
      { title: '时间', dataIndex: 'created_at', key: 'created_at', render: (value: string) => new Date(value).toLocaleString() },
      { title: '交易对', dataIndex: 'symbol', key: 'symbol', render: (value: string) => symbolLabelMap.get(value) ?? value },
      {
        title: '方向',
        dataIndex: 'side',
        key: 'side',
        render: (value: string) => <Tag color={value === 'long' ? 'green' : 'red'}>{value}</Tag>,
      },
      { title: '数量', dataIndex: 'size', key: 'size', render: (value: string) => formatAmount(value, 4) },
      { title: '成交价', dataIndex: 'exec_price', key: 'exec_price', render: (value: string) => formatAmount(value, 2) },
      { title: '杠杆', dataIndex: 'leverage', key: 'leverage', render: (value: number) => `${value}x` },
      { title: '手续费', dataIndex: 'fee', key: 'fee', render: (value: string) => formatAmount(value) },
      {
        title: '已实现盈亏',
        dataIndex: 'realized_pnl',
        key: 'realized_pnl',
        render: (value: string) => (
          <Typography.Text style={{ color: Number(value) >= 0 ? '#2ec9b0' : '#ff6b6b' }}>{formatAmount(value)}</Typography.Text>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (value: string) => <Tag color={value === 'filled' ? 'cyan' : 'default'}>{value}</Tag>,
      },
    ],
    [symbolLabelMap],
  )

  return (
    <div>
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
                <Typography.Text type="secondary">Oracle</Typography.Text>
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
                  <Tag color="blue">Live Market Feed</Tag>
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
                  initialValues={{ side: 'long', leverage: 10, reduce_only: false }}
                  onFinish={(values) => orderMutation.mutate(values)}
                >
                  <Form.Item label="方向" name="side">
                    <Segmented block options={[{ label: '做多', value: 'long' }, { label: '做空', value: 'short' }]} />
                  </Form.Item>
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
                    <InputNumber min={1} max={currentSymbol?.max_leverage ?? 40} style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item label={`杠杆滑块 ${selectedLeverage}x`}>
                    <Slider min={1} max={currentSymbol?.max_leverage ?? 40} value={selectedLeverage} onChange={(value) => orderForm.setFieldValue('leverage', value)} />
                  </Form.Item>
                  <Form.Item label="只减仓" name="reduce_only" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Card size="small" style={{ marginBottom: 16 }}>
                    <Descriptions size="small" column={1}>
                      <Descriptions.Item label="方向">{selectedSide === 'long' ? '做多 Long' : '做空 Short'}</Descriptions.Item>
                      <Descriptions.Item label="名义价值">{formatAmount(estimatedNotional, 2)} USDC</Descriptions.Item>
                      <Descriptions.Item label="预估保证金">{formatAmount(estimatedMargin)} USDC</Descriptions.Item>
                      <Descriptions.Item label="预估手续费">{formatAmount(estimatedFee)} USDC</Descriptions.Item>
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
            {authenticated && topPosition ? (
              <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
                <Col xs={24} md={8}>
                  <Card size="small" className="rg-glass-card">
                    <Typography.Text type="secondary">未实现盈亏</Typography.Text>
                    <Typography.Title level={4} style={{ margin: 0, color: Number(topPosition.unrealized_pnl) >= 0 ? '#2ec9b0' : '#ff6b6b' }}>
                      {formatAmount(topPosition.unrealized_pnl)} USDC
                    </Typography.Title>
                  </Card>
                </Col>
                <Col xs={24} md={8}>
                  <Card size="small" className="rg-glass-card">
                    <Typography.Text type="secondary">保证金率</Typography.Text>
                    <Typography.Title level={4} style={{ margin: 0 }}>
                      {formatAmount(marginRatio, 2)}%
                    </Typography.Title>
                  </Card>
                </Col>
                <Col xs={24} md={8}>
                  <Card size="small" className="rg-glass-card">
                    <Typography.Text type="secondary">距清算</Typography.Text>
                    <Typography.Title level={4} style={{ margin: 0, color: liquidationDistance > 8 ? '#2ec9b0' : '#fbbf24' }}>
                      {formatAmount(liquidationDistance, 2)}%
                    </Typography.Title>
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
                    <Table
                      rowKey="id"
                      loading={positionsQuery.isLoading}
                      dataSource={positionsQuery.data ?? []}
                      columns={positionColumns}
                      pagination={false}
                      locale={{ emptyText: '当前无持仓' }}
                      scroll={{ x: 960 }}
                    />
                  ),
                },
                {
                  key: 'open-orders',
                  label: 'Open Orders',
                  children: !authenticated ? (
                    <Typography.Text type="secondary">登录后可查看挂单。</Typography.Text>
                  ) : (
                    <Table
                      rowKey="id"
                      loading={openOrdersQuery.isLoading}
                      dataSource={openOrdersQuery.data ?? []}
                      columns={orderColumns}
                      pagination={false}
                      locale={{ emptyText: '当前无挂单' }}
                      scroll={{ x: 1080 }}
                    />
                  ),
                },
                {
                  key: 'trade-history',
                  label: 'Trade History',
                  children: !authenticated ? (
                    <Typography.Text type="secondary">登录后可查看成交记录。</Typography.Text>
                  ) : (
                    <Table
                      rowKey="id"
                      loading={tradesQuery.isLoading}
                      dataSource={tradesQuery.data ?? []}
                      columns={tradeColumns}
                      pagination={false}
                      locale={{ emptyText: '暂无成交记录' }}
                      scroll={{ x: 1080 }}
                    />
                  ),
                },
                {
                  key: 'order-history',
                  label: 'Order History',
                  children: !authenticated ? (
                    <Typography.Text type="secondary">登录后可查看订单历史。</Typography.Text>
                  ) : (
                    <Table
                      rowKey="id"
                      loading={ordersQuery.isLoading}
                      dataSource={ordersQuery.data ?? []}
                      columns={orderColumns}
                      pagination={false}
                      locale={{ emptyText: '暂无订单记录' }}
                      scroll={{ x: 1080 }}
                    />
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
