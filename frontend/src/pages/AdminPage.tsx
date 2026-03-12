import { Typography, Card, Row, Col } from 'antd'

export default function AdminPage() {
  return (
    <div>
      <Typography.Title level={3}>系统管理</Typography.Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={12}>
          <Card title="系统风险">
            <Typography.Text type="secondary">风险概览 — 里程碑 4 实现</Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card title="对冲状态">
            <Typography.Text type="secondary">对冲监控 — 里程碑 4 实现</Typography.Text>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
