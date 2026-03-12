import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { App as AntdApp, ConfigProvider, theme } from 'antd'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useEffect } from 'react'
import AppLayout from './components/layout/AppLayout'
import TradePage from './pages/TradePage'
import AccountPage from './pages/AccountPage'
import { useThemeStore } from './stores/themeStore'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5000,
    },
  },
})

export default function App() {
  const { mode } = useThemeStore()

  useEffect(() => {
    document.body.classList.toggle('light-theme', mode === 'light')
  }, [mode])

  const isDark = mode === 'dark'

  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider
        theme={{
          algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
          token: {
            colorPrimary: isDark ? '#20c9b5' : '#1677ff',
            colorSuccess: '#20c997',
            colorError: '#ff6b6b',
            borderRadius: 10,
            fontFamily: '"IBM Plex Sans", "PingFang SC", "Noto Sans SC", sans-serif',
            colorBgBase: isDark ? '#08131b' : '#f3f7fb',
            colorBgContainer: isDark ? '#0d1b24' : '#ffffff',
            colorText: isDark ? '#edf6ff' : '#10202d',
            colorTextSecondary: isDark ? '#8ca3b8' : '#526476',
            colorBorderSecondary: isDark ? '#17303d' : '#d9e3ec',
          },
          components: {
            Layout: {
              headerBg: isDark ? 'rgba(10, 24, 33, 0.88)' : 'rgba(255, 255, 255, 0.88)',
              bodyBg: 'transparent',
              footerBg: 'transparent',
              siderBg: 'transparent',
            },
            Card: {
              colorBgContainer: isDark ? 'rgba(12, 27, 36, 0.88)' : 'rgba(255,255,255,0.9)',
              boxShadowTertiary: isDark
                ? '0 16px 40px rgba(0,0,0,0.28)'
                : '0 10px 30px rgba(15,23,42,0.08)',
            },
            Table: {
              headerBg: isDark ? '#102431' : '#f8fafc',
              rowHoverBg: isDark ? '#102634' : '#f1f5f9',
            },
            Segmented: {
              trackBg: isDark ? '#10212d' : '#edf2f7',
            },
          },
        }}
      >
        <AntdApp>
          <BrowserRouter>
            <Routes>
              <Route element={<AppLayout />}>
                <Route path="/" element={<TradePage />} />
                <Route path="/account" element={<AccountPage />} />
              </Route>
            </Routes>
          </BrowserRouter>
        </AntdApp>
      </ConfigProvider>
    </QueryClientProvider>
  )
}
