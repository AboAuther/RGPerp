import { Typography, Card, Row, Col } from 'antd'

export default function AccountPage() {
  return (
    <div>
      <Typography.Title level={3}>资产总览</Typography.Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <Card title="账户余额">
            <Typography.Text type="secondary">余额信息 — 里程碑 2 实现</Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card title="充值">
            <Typography.Text type="secondary">充值功能 — 里程碑 2 实现</Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card title="提现">
            <Typography.Text type="secondary">提现功能 — 里程碑 2 实现</Typography.Text>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
