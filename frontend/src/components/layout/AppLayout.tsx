import { Button, Layout, Menu, Space, Typography, theme } from 'antd'
import {
  DashboardOutlined,
  BulbOutlined,
  LineChartOutlined,
  MoonOutlined,
  WalletOutlined,
} from '@ant-design/icons'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useThemeStore } from '../../stores/themeStore'
import BrandLogo from '../brand/BrandLogo'

const { Header, Content, Footer } = Layout

const menuItems = [
  { key: '/app', icon: <LineChartOutlined />, label: '交易' },
  { key: '/app/account', icon: <WalletOutlined />, label: '资产' },
  { key: '/app/admin', icon: <DashboardOutlined />, label: '管理' },
]

export default function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { token } = theme.useToken()
  const { mode, toggleMode } = useThemeStore()
  const isDark = mode === 'dark'
  const selectedMenuKey = location.pathname.startsWith('/app/account')
    ? '/app/account'
    : location.pathname.startsWith('/app/admin')
      ? '/app/admin'
      : '/app'

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
        <button
          type="button"
          onClick={() => navigate('/')}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 12,
            marginRight: 24,
            padding: 0,
            border: 'none',
            background: 'transparent',
            cursor: 'pointer',
            color: 'inherit',
          }}
          aria-label="返回首页"
        >
          <BrandLogo size={34} />
          <Typography.Title level={4} style={{ margin: 0, whiteSpace: 'nowrap' }}>
            RGPerp
          </Typography.Title>
        </button>
        <Menu
          mode="horizontal"
          selectedKeys={[selectedMenuKey]}
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
