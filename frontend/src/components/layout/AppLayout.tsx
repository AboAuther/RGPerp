import { Layout, Menu, Typography, theme } from 'antd'
import {
  LineChartOutlined,
  WalletOutlined,
  HistoryOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'

const { Header, Content, Footer } = Layout

const menuItems = [
  { key: '/', icon: <LineChartOutlined />, label: '交易' },
  { key: '/account', icon: <WalletOutlined />, label: '资产' },
  { key: '/history', icon: <HistoryOutlined />, label: '历史' },
  { key: '/admin', icon: <SettingOutlined />, label: '管理' },
]

export default function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { token: { colorBgContainer } } = theme.useToken()

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          background: colorBgContainer,
          borderBottom: '1px solid #f0f0f0',
          padding: '0 24px',
        }}
      >
        <Typography.Title level={4} style={{ margin: '0 24px 0 0', whiteSpace: 'nowrap' }}>
          PerpExchange
        </Typography.Title>
        <Menu
          mode="horizontal"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ flex: 1, border: 'none' }}
        />
      </Header>
      <Content style={{ padding: '24px', background: colorBgContainer }}>
        <Outlet />
      </Content>
      <Footer style={{ textAlign: 'center', color: '#999' }}>
        PerpExchange &copy; {new Date().getFullYear()}
      </Footer>
    </Layout>
  )
}
