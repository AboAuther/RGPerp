import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { ConfigProvider, App as AntdApp } from 'antd'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import AppLayout from './components/layout/AppLayout'
import TradePage from './pages/TradePage'
import AccountPage from './pages/AccountPage'
import HistoryPage from './pages/HistoryPage'
import AdminPage from './pages/AdminPage'

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
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider
        theme={{
          token: {
            colorPrimary: '#1677ff',
            borderRadius: 6,
          },
        }}
      >
        <AntdApp>
          <BrowserRouter>
            <Routes>
              <Route element={<AppLayout />}>
                <Route path="/" element={<TradePage />} />
                <Route path="/account" element={<AccountPage />} />
                <Route path="/history" element={<HistoryPage />} />
                <Route path="/admin" element={<AdminPage />} />
              </Route>
            </Routes>
          </BrowserRouter>
        </AntdApp>
      </ConfigProvider>
    </QueryClientProvider>
  )
}
