import { Typography, Card } from 'antd'

export default function HistoryPage() {
  return (
    <div>
      <Typography.Title level={3}>交易历史</Typography.Title>
      <Card>
        <Typography.Text type="secondary">订单与成交记录 — 里程碑 3 实现</Typography.Text>
      </Card>
    </div>
  )
}
