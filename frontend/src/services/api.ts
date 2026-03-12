import axios from 'axios'
import type { ApiResponse } from '../types'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/'
    }
    return Promise.reject(error)
  },
)

export async function get<T>(url: string, params?: Record<string, unknown>): Promise<ApiResponse<T>> {
  const res = await api.get<ApiResponse<T>>(url, { params })
  return res.data
}

export async function post<T>(
  url: string,
  data?: unknown,
  headers?: Record<string, string>,
): Promise<ApiResponse<T>> {
  const res = await api.post<ApiResponse<T>>(url, data, { headers })
  return res.data
}

export default api
