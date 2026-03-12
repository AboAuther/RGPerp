import { create } from 'zustand'

interface AuthState {
  token: string | null
  walletAddress: string | null
  setAuth: (token: string, walletAddress: string) => void
  logout: () => void
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem('token'),
  walletAddress: localStorage.getItem('walletAddress'),

  setAuth: (token: string, walletAddress: string) => {
    localStorage.setItem('token', token)
    localStorage.setItem('walletAddress', walletAddress)
    set({ token, walletAddress })
  },

  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('walletAddress')
    set({ token: null, walletAddress: null })
  },

  isAuthenticated: () => !!get().token,
}))
