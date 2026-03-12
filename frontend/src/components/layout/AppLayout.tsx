import { Button, Layout, Menu, Space, Typography, theme } from 'antd'
import {
  BulbOutlined,
  LineChartOutlined,
  MoonOutlined,
  WalletOutlined,
} from '@ant-design/icons'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useThemeStore } from '../../stores/themeStore'

const { Header, Content, Footer } = Layout

const menuItems = [
  { key: '/', icon: <LineChartOutlined />, label: '交易' },
  { key: '/account', icon: <WalletOutlined />, label: '资产' },
]

export default function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { token } = theme.useToken()
  const { mode, toggleMode } = useThemeStore()
  const isDark = mode === 'dark'

  return (
    <Layout style={{ minHeight: '100vh', background: 'transparent' }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          background: token.colorBgContainer,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          padding: '0 24px',
          backdropFilter: 'blur(16px)',
          position: 'sticky',
          top: 0,
          zIndex: 20,
        }}
        className="rg-glass-card"
      >
        <Space size={12} style={{ marginRight: 24 }}>
          <div
            style={{
              width: 14,
              height: 14,
              borderRadius: 999,
              background: 'linear-gradient(135deg, #20c9b5 0%, #33d69f 100%)',
              boxShadow: '0 0 18px rgba(32, 201, 181, 0.65)',
            }}
          />
          <Typography.Title level={4} style={{ margin: 0, whiteSpace: 'nowrap' }}>
            RGPerp
          </Typography.Title>
        </Space>
        <Menu
          mode="horizontal"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ flex: 1, border: 'none', background: 'transparent' }}
        />
        <Button
          type="text"
          icon={isDark ? <BulbOutlined /> : <MoonOutlined />}
          onClick={() => toggleMode()}
        >
          {isDark ? 'Light' : 'Dark'}
        </Button>
      </Header>
      <Content style={{ padding: '24px', background: 'transparent', maxWidth: 1600, width: '100%', margin: '0 auto' }}>
        <Outlet />
      </Content>
      <Footer style={{ textAlign: 'center', color: token.colorTextSecondary, background: 'transparent' }}>
        RGPerp &copy; {new Date().getFullYear()}
      </Footer>
    </Layout>
  )
}
