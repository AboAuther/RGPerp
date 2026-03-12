import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  Card,
  Col,
  Descriptions,
  Empty,
  Row,
  Segmented,
  Select,
  Skeleton,
  Typography,
} from 'antd'
import { useQuery } from '@tanstack/react-query'
import { get } from '../services/api'
import type { KlineItem, MarketTicker, SymbolInfo } from '../types'
import KlineChart from '../components/trading/KlineChart'

export default function TradePage() {
  const [symbol, setSymbol] = useState<string>('BTC-PERP')
  const [interval, setInterval] = useState<'1m' | '5m' | '15m' | '1h'>('1m')

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

  const marketOptions = useMemo(
    () =>
      (marketsQuery.data ?? []).map((item) => ({
        label: item.name,
        value: item.name,
      })),
    [marketsQuery.data],
  )

  const currentSymbol = (marketsQuery.data ?? []).find((item) => item.name === symbol)

  return (
    <div>
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
          <Card title="K 线图表" style={{ minHeight: 400 }}>
            <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
              <Col>
                <Typography.Text type="secondary">TradingView Lightweight Charts</Typography.Text>
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
              <KlineChart data={klinesQuery.data} />
            ) : (
              <Empty description="暂无K线数据" />
            )}
            <Typography.Text>
              当前交易对: {currentSymbol?.name ?? '-'}（{currentSymbol?.base_asset ?? '-'} /{' '}
              {currentSymbol?.quote_asset ?? '-'}）
            </Typography.Text>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="下单面板" style={{ minHeight: 400 }}>
            <Alert
              type="info"
              showIcon
              message="里程碑3开发中"
              description="下一步将实现 CFD 市价单执行、保证金与仓位变更。"
            />
          </Card>
        </Col>
        <Col xs={24}>
          <Card title="当前持仓">
            <Typography.Text type="secondary">持仓列表与成交历史将在下一批接口完成后联调。</Typography.Text>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
