import { Alert, Button, Card, Col, Row, Space, Table, Tag, Typography, message } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { get, getErrorMessage, post } from '../services/api'
import type {
  AdminAlertItem,
  AdminHedgeTaskItem,
  AdminLiquidationItem,
  AdminOverview,
  AdminRiskSnapshotItem,
} from '../types'

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

function levelTag(level: string) {
  switch (level) {
    case 'error':
      return 'red'
    case 'warning':
      return 'gold'
    default:
      return 'blue'
  }
}

function hedgeStatusColor(status: string) {
  switch (status) {
    case 'completed':
      return 'green'
    case 'buffered':
      return 'blue'
    case 'retrying':
      return 'gold'
    case 'failed':
      return 'red'
    case 'pending':
      return 'cyan'
    default:
      return 'default'
  }
}

function formatAdminPair(symbol?: string) {
  if (!symbol) {
    return '--'
  }
  return symbol.replace('-PERP', '/USDC')
}

function formatDriftSummary(overview?: AdminOverview) {
  const rows = overview?.drift_by_symbol ?? []
  if (rows.length === 0) {
    return '暂无快照'
  }
  return rows
    .map((item) => `${formatAdminPair(item.symbol)} ${formatAmount(item.drift, 4)}`)
    .join(' / ')
}

export default function AdminPage() {
  const queryClient = useQueryClient()
  const overviewQuery = useQuery({
    queryKey: ['admin-overview'],
    queryFn: async () => (await get<AdminOverview>('/admin/overview')).data!,
    refetchInterval: 4000,
  })

  const alertsQuery = useQuery({
    queryKey: ['admin-alerts'],
    queryFn: async () => (await get<{ items: AdminAlertItem[] }>('/admin/alerts', { limit: 8 })).data?.items ?? [],
    refetchInterval: 4000,
  })

  const hedgeTasksQuery = useQuery({
    queryKey: ['admin-hedge-tasks'],
    queryFn: async () => (await get<{ items: AdminHedgeTaskItem[] }>('/admin/hedge-tasks', { limit: 50 })).data?.items ?? [],
    refetchInterval: 4000,
  })

  const riskSnapshotsQuery = useQuery({
    queryKey: ['admin-risk-snapshots'],
    queryFn: async () => (await get<{ items: AdminRiskSnapshotItem[] }>('/admin/risk-snapshots', { limit: 10 })).data?.items ?? [],
    refetchInterval: 4000,
  })

  const liquidationsQuery = useQuery({
    queryKey: ['admin-liquidations'],
    queryFn: async () => (await get<{ items: AdminLiquidationItem[] }>('/admin/liquidations', { limit: 10 })).data?.items ?? [],
    refetchInterval: 4000,
  })

  const retryMutation = useMutation({
    mutationFn: async (taskId: number) => post(`/admin/hedge-tasks/${taskId}/retry`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-overview'] })
      void queryClient.invalidateQueries({ queryKey: ['admin-alerts'] })
      void queryClient.invalidateQueries({ queryKey: ['admin-hedge-tasks'] })
      message.success('已重新提交对冲任务')
    },
    onError: (error) => {
      message.error(getErrorMessage(error, '重试对冲任务失败'))
    },
  })

  const overview = overviewQuery.data

  return (
    <div className="rg-app-page rg-app-page--admin">
      <Typography.Title level={3}>系统管理</Typography.Title>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} md={12} xl={6} style={{ display: 'flex' }}>
          <Card className="rg-glass-card rg-admin-summary-card" style={{ width: '100%' }}>
            <Typography.Text type="secondary">交易中市场</Typography.Text>
            <Typography.Title level={3} style={{ margin: 0 }}>
              {overview?.trading_symbols ?? '--'}
            </Typography.Title>
            <Typography.Text type="secondary">当前 open positions: {overview?.open_positions ?? '--'}</Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6} style={{ display: 'flex' }}>
          <Card className="rg-glass-card rg-admin-summary-card" style={{ width: '100%' }}>
            <Typography.Text type="secondary">风险账户</Typography.Text>
            <Typography.Title level={3} style={{ margin: 0 }}>
              {(overview?.accounts_at_risk ?? 0) + (overview?.accounts_reduce_only ?? 0) + (overview?.accounts_liquidating ?? 0)}
            </Typography.Title>
            <Typography.Text type="secondary">
              At Risk {overview?.accounts_at_risk ?? 0} / Reduce Only {overview?.accounts_reduce_only ?? 0} / Liquidating {overview?.accounts_liquidating ?? 0}
            </Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6} style={{ display: 'flex' }}>
          <Card className="rg-glass-card rg-admin-summary-card" style={{ width: '100%' }}>
            <Typography.Text type="secondary">对冲任务</Typography.Text>
            <Typography.Title level={3} style={{ margin: 0 }}>
              {(overview?.pending_hedges ?? 0) + (overview?.buffered_hedges ?? 0) + (overview?.retrying_hedges ?? 0) + (overview?.failed_hedges ?? 0)}
            </Typography.Title>
            <Typography.Text type="secondary">
              Pending {overview?.pending_hedges ?? 0} / Buffered {overview?.buffered_hedges ?? 0} / Retrying {overview?.retrying_hedges ?? 0} / Failed {overview?.failed_hedges ?? 0}
            </Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={12} xl={6} style={{ display: 'flex' }}>
          <Card className="rg-glass-card rg-admin-summary-card" style={{ width: '100%' }}>
            <Typography.Text type="secondary">各交易对净敞口偏差</Typography.Text>
            <Typography.Title level={3} style={{ margin: 0 }}>
              {overview?.unhealthy_symbols ?? 0}
            </Typography.Title>
            <Typography.Text type="secondary">
              {formatDriftSummary(overview)}
            </Typography.Text>
            <Typography.Text type="secondary">
              异常交易对 {overview?.unhealthy_symbols ?? 0} · 最近快照 {overview?.last_snapshot_at ? new Date(overview.last_snapshot_at).toLocaleString() : '--'}
            </Typography.Text>
          </Card>
        </Col>
      </Row>

      <Card title="风险告警" className="rg-glass-card" style={{ marginBottom: 16 }}>
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          {(alertsQuery.data ?? []).length === 0 ? (
            <Alert type="success" showIcon message="当前没有高优先级风险告警" />
          ) : (
            alertsQuery.data?.map((item, index) => (
              <Alert
                key={`${item.category}-${item.created_at}-${index}`}
                type={item.level === 'error' ? 'error' : 'warning'}
                showIcon
                message={
                  <Space size={8}>
                    <Tag color={levelTag(item.level)}>{item.category}</Tag>
                    <span>{item.title}</span>
                    {item.symbol ? <Tag>{formatAdminPair(item.symbol)}</Tag> : null}
                  </Space>
                }
                description={`${item.detail} · ${new Date(item.created_at).toLocaleString()}`}
              />
            ))
          )}
        </Space>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} xl={12}>
          <Card title="对冲任务" className="rg-glass-card">
            <Table
              rowKey="id"
              loading={hedgeTasksQuery.isLoading}
              dataSource={hedgeTasksQuery.data ?? []}
              pagination={false}
              scroll={{ x: 980 }}
              columns={[
                { title: '时间', dataIndex: 'updated_at', key: 'updated_at', width: 168, render: (v: string) => new Date(v).toLocaleString() },
                { title: '交易对', dataIndex: 'symbol', key: 'symbol', width: 96, render: (value: string) => formatAdminPair(value) },
                { title: '触发', dataIndex: 'trigger_type', key: 'trigger_type', width: 84 },
                { title: '目标', dataIndex: 'target_hedge_position', key: 'target_hedge_position', width: 92, render: (v: string) => formatAmount(v, 4) },
                { title: '当前', dataIndex: 'current_hedge_position', key: 'current_hedge_position', width: 92, render: (v: string) => formatAmount(v, 4) },
                { title: '偏差', dataIndex: 'drift', key: 'drift', width: 92, render: (v: string) => formatAmount(v, 4) },
                { title: '状态', dataIndex: 'status', key: 'status', width: 90, render: (v: string) => <Tag color={hedgeStatusColor(v)}>{v}</Tag> },
                { title: '外部单', dataIndex: 'last_order_status', key: 'last_order_status', width: 90, render: (v: string) => <Tag>{v || '-'}</Tag> },
                {
                  title: '操作',
                  key: 'actions',
                  width: 96,
                  render: (_: unknown, record: AdminHedgeTaskItem) =>
                    record.status === 'failed' ? (
                      <Button
                        size="small"
                        onClick={() => retryMutation.mutate(record.id)}
                        loading={retryMutation.isPending}
                      >
                        重试
                      </Button>
                    ) : '--',
                },
              ]}
              expandable={{
                expandedRowRender: (record) => (
                  <Space direction="vertical" size={4}>
                    <Typography.Text type="secondary">错误信息：{record.error_message || '--'}</Typography.Text>
                    <Typography.Text type="secondary">
                      外部方向：{record.last_order_side || '--'} / 数量：{formatAmount(record.last_order_size, 6)} / 价格：{formatAmount(record.last_order_price, 2)} / 自动重试次数：{record.last_order_retry_count ?? 0}
                    </Typography.Text>
                  </Space>
                ),
                rowExpandable: (record) => Boolean(record.error_message || record.last_order_side),
              }}
            />
          </Card>
        </Col>
        <Col xs={24} xl={12}>
          <Card title="风险快照" className="rg-glass-card">
            <Table
              rowKey="id"
              loading={riskSnapshotsQuery.isLoading}
              dataSource={riskSnapshotsQuery.data ?? []}
              pagination={false}
              scroll={{ x: 980 }}
              columns={[
                { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 168, render: (v: string) => new Date(v).toLocaleString() },
                { title: '交易对', dataIndex: 'symbol', key: 'symbol', width: 96, render: (value: string) => formatAdminPair(value) },
                { title: '内部净仓', dataIndex: 'net_position', key: 'net_position', width: 96, render: (v: string) => formatAmount(v, 4) },
                { title: '外部对冲', dataIndex: 'external_hedge_position', key: 'external_hedge_position', width: 96, render: (v: string) => formatAmount(v, 4) },
                { title: '偏差', dataIndex: 'drift', key: 'drift', width: 96, render: (v: string) => formatAmount(v, 4) },
                { title: '健康', dataIndex: 'hedge_healthy', key: 'hedge_healthy', width: 80, render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? 'healthy' : 'alert'}</Tag> },
                { title: 'OI', dataIndex: 'total_open_interest', key: 'total_open_interest', width: 90, render: (v: string) => formatAmount(v, 2) },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24}>
          <Card title="强平记录" className="rg-glass-card">
            <Table
              rowKey="id"
              loading={liquidationsQuery.isLoading}
              dataSource={liquidationsQuery.data ?? []}
              pagination={false}
              scroll={{ x: 1200 }}
              columns={[
                { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 168, render: (v: string) => new Date(v).toLocaleString() },
                { title: '用户', dataIndex: 'user_id', key: 'user_id', width: 72 },
                { title: '交易对', dataIndex: 'symbol', key: 'symbol', width: 96, render: (value: string) => formatAdminPair(value) },
                { title: '方向', dataIndex: 'side', key: 'side', width: 76, render: (v: string) => <Tag color={v === 'long' ? 'green' : 'red'}>{v}</Tag> },
                { title: '数量', dataIndex: 'size', key: 'size', width: 84, render: (v: string) => formatAmount(v, 4) },
                { title: '开仓价', dataIndex: 'entry_price', key: 'entry_price', width: 96, render: (v: string) => formatAmount(v, 2) },
                { title: '标记价', dataIndex: 'mark_price', key: 'mark_price', width: 96, render: (v: string) => formatAmount(v, 2) },
                { title: '执行价', dataIndex: 'execution_price', key: 'execution_price', width: 96, render: (v: string) => formatAmount(v, 2) },
                { title: '已实现盈亏', dataIndex: 'realized_pnl', key: 'realized_pnl', width: 108, render: (v: string) => formatAmount(v, 4) },
                { title: '类型', dataIndex: 'type', key: 'type', width: 82, render: (v: string) => <Tag>{v}</Tag> },
                { title: '状态', dataIndex: 'status', key: 'status', width: 82, render: (v: string) => <Tag color={v === 'completed' ? 'green' : 'gold'}>{v}</Tag> },
              ]}
            />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
