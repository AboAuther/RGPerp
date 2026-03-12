import { Typography, Row, Col, Card } from 'antd'

export default function TradePage() {
  return (
    <div>
      <Typography.Title level={3}>BTC-PERP 永续合约</Typography.Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card title="K 线图表" style={{ minHeight: 400 }}>
            <Typography.Text type="secondary">TradingView 图表 — 里程碑 3 实现</Typography.Text>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="下单面板" style={{ minHeight: 400 }}>
            <Typography.Text type="secondary">订单表单 — 里程碑 3 实现</Typography.Text>
          </Card>
        </Col>
        <Col xs={24}>
          <Card title="当前持仓">
            <Typography.Text type="secondary">持仓列表 — 里程碑 3 实现</Typography.Text>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
