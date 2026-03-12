import { create } from 'zustand'

type ThemeMode = 'dark' | 'light'

interface ThemeState {
  mode: ThemeMode
  setMode: (mode: ThemeMode) => void
  toggleMode: () => void
}

const initialMode = (localStorage.getItem('themeMode') as ThemeMode | null) ?? 'dark'

export const useThemeStore = create<ThemeState>((set, get) => ({
  mode: initialMode,
  setMode: (mode) => {
    localStorage.setItem('themeMode', mode)
    set({ mode })
  },
  toggleMode: () => {
    const next = get().mode === 'dark' ? 'light' : 'dark'
    localStorage.setItem('themeMode', next)
    set({ mode: next })
  },
}))
