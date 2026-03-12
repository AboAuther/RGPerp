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
  Row,
  Segmented,
  Select,
  Skeleton,
  Space,
  Switch,
  Table,
  Typography,
  message,
} from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { get, post } from '../services/api'
import type { Account, AuthChallenge, AuthLogin, KlineItem, MarketTicker, OrderExecution, Position, SymbolInfo } from '../types'
import KlineChart from '../components/trading/KlineChart'
import { useThemeStore } from '../stores/themeStore'
import { useAuthStore } from '../stores/authStore'

export default function TradePage() {
  const [symbol, setSymbol] = useState<string>('BTC-PERP')
  const [interval, setInterval] = useState<'1m' | '5m' | '15m' | '1h'>('1m')
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

  const klinesQuery = useQuery({
    queryKey: ['klines', symbol, interval],
    queryFn: async () =>
      (
        await get<KlineItem[]>(`/markets/${symbol}/klines`, {
          interval,
          limit: 200,
        })
      ).data ?? [],
    enabled: Boolean(symbol),
    refetchInterval: 5_000,
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
      void messageApi.success('市价单已成交')
    },
    onError: (error) => {
      void messageApi.error(error instanceof Error ? error.message : '下单失败')
    },
  })

  const marketOptions = useMemo(
    () =>
      (marketsQuery.data ?? []).map((item) => ({
        label: item.name,
        value: item.name,
      })),
    [marketsQuery.data],
  )

  const currentSymbol = (marketsQuery.data ?? []).find((item) => item.name === symbol)
  const positionColumns = useMemo(
    () => [
      { title: '方向', dataIndex: 'side', key: 'side' },
      { title: '数量', dataIndex: 'size', key: 'size' },
      { title: '开仓价', dataIndex: 'entry_price', key: 'entry_price' },
      { title: '标记价', dataIndex: 'mark_price', key: 'mark_price' },
      { title: '保证金', dataIndex: 'margin', key: 'margin' },
      { title: '未实现盈亏', dataIndex: 'unrealized_pnl', key: 'unrealized_pnl' },
      { title: '清算价', dataIndex: 'liquidation_price', key: 'liquidation_price' },
    ],
    [],
  )

  return (
    <div>
      {contextHolder}
      <Card
        style={{ marginBottom: 16, border: 'none' }}
        styles={{ body: { padding: 18 } }}
        className="rg-glass-card"
      >
        <Row gutter={[20, 12]} align="middle" justify="space-between">
          <Col>
            <Space direction="vertical" size={2}>
              <Space size={10}>
                <Typography.Title level={3} style={{ margin: 0 }}>
                  BTC-USDC
                </Typography.Title>
                <Button size="small" type="primary" ghost>
                  40x
                </Button>
              </Space>
              <Typography.Text type="secondary">Chart feed aligned to BTC/USDC market</Typography.Text>
            </Space>
          </Col>
          <Col>
            <Space size={24} wrap>
              <div>
                <Typography.Text type="secondary">Mark</Typography.Text>
                <Typography.Title level={4} style={{ margin: 0, color: '#2ec9b0' }}>
                  {tickerQuery.data?.mark_price ?? '--'}
                </Typography.Title>
              </div>
              <div>
                <Typography.Text type="secondary">Oracle</Typography.Text>
                <Typography.Title level={4} style={{ margin: 0 }}>
                  {tickerQuery.data?.index_price ?? '--'}
                </Typography.Title>
              </div>
              <div>
                <Typography.Text type="secondary">Spread</Typography.Text>
                <Typography.Title level={4} style={{ margin: 0 }}>
                  {tickerQuery.data
                    ? `${(
                        Number(tickerQuery.data.best_ask) - Number(tickerQuery.data.best_bid)
                      ).toFixed(2)}`
                    : '--'}
                </Typography.Title>
              </div>
            </Space>
          </Col>
        </Row>
      </Card>

      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col>
          <Typography.Title level={3} style={{ marginBottom: 0 }}>
            永续合约交易
          </Typography.Title>
        </Col>
        <Col>
          <Select
            value={symbol}
            options={marketOptions}
            onChange={(v) => setSymbol(v)}
            style={{ minWidth: 180 }}
            loading={marketsQuery.isLoading}
            placeholder="选择交易对"
          />
        </Col>
      </Row>

      <Card style={{ marginBottom: 16 }}>
        {tickerQuery.isLoading ? (
          <Skeleton active paragraph={{ rows: 1 }} />
        ) : tickerQuery.data ? (
          <Descriptions size="small" column={{ xs: 1, sm: 2, md: 4 }}>
            <Descriptions.Item label="标记价格">{tickerQuery.data.mark_price}</Descriptions.Item>
            <Descriptions.Item label="指数价格">{tickerQuery.data.index_price}</Descriptions.Item>
            <Descriptions.Item label="最优买/卖">
              {tickerQuery.data.best_bid} / {tickerQuery.data.best_ask}
            </Descriptions.Item>
            <Descriptions.Item label="更新时间">
              {new Date(tickerQuery.data.timestamp * 1000).toLocaleString()}
            </Descriptions.Item>
          </Descriptions>
        ) : (
          <Empty description="暂无行情数据" />
        )}
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card title="K 线图表" style={{ minHeight: 400 }} className="rg-glass-card">
            <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
              <Col>
                <Space size={10}>
                  <Typography.Text strong>BTCUSD</Typography.Text>
                  <Typography.Text type="secondary">Hyperliquid style</Typography.Text>
                </Space>
              </Col>
              <Col>
                <Segmented
                  value={interval}
                  options={['1m', '5m', '15m', '1h']}
                  onChange={(v) => setInterval(v as '1m' | '5m' | '15m' | '1h')}
                />
              </Col>
            </Row>
            {klinesQuery.isLoading ? (
              <Skeleton active paragraph={{ rows: 8 }} />
            ) : klinesQuery.data && klinesQuery.data.length > 0 ? (
              <KlineChart data={klinesQuery.data} dark={isDark} />
            ) : (
              <Empty description="暂无K线数据" />
            )}
            <Descriptions
              size="small"
              column={{ xs: 1, sm: 2, md: 4 }}
              style={{ marginTop: 16 }}
            >
              <Descriptions.Item label="交易标的">{currentSymbol?.name ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="图表来源">BTC / USDC</Descriptions.Item>
              <Descriptions.Item label="最优买">{tickerQuery.data?.best_bid ?? '--'}</Descriptions.Item>
              <Descriptions.Item label="最优卖">{tickerQuery.data?.best_ask ?? '--'}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="下单面板" style={{ minHeight: 400 }} className="rg-glass-card">
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
                <Descriptions size="small" column={1}>
                  <Descriptions.Item label="可用余额">{accountQuery.data?.available_balance ?? '--'} USDC</Descriptions.Item>
                  <Descriptions.Item label="锁定保证金">{accountQuery.data?.locked_balance ?? '--'} USDC</Descriptions.Item>
                </Descriptions>
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
                    <Input placeholder="例如 10" />
                  </Form.Item>
                  <Form.Item label="只减仓" name="reduce_only" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Button type="primary" htmlType="submit" block loading={orderMutation.isPending}>
                    市价下单
                  </Button>
                </Form>

                {latestExecution ? (
                  <Card size="small">
                    <Descriptions size="small" column={1}>
                      <Descriptions.Item label="成交价">{latestExecution.order.exec_price}</Descriptions.Item>
                      <Descriptions.Item label="手续费">{latestExecution.order.fee}</Descriptions.Item>
                      <Descriptions.Item label="可用余额">{latestExecution.account.available_balance}</Descriptions.Item>
                    </Descriptions>
                  </Card>
                ) : null}
              </Space>
            )}
          </Card>
        </Col>
        <Col xs={24}>
          <Card title="当前持仓" className="rg-glass-card">
            {!authenticated ? (
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
            )}
          </Card>
        </Col>
      </Row>
    </div>
  )
}
