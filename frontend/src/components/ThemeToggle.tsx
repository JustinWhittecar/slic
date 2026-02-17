import { useState, useEffect } from 'react'

type Theme = 'system' | 'light' | 'dark'

function getEffectiveTheme(theme: Theme): 'light' | 'dark' {
  if (theme === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return theme
}

function applyTheme(theme: Theme) {
  const effective = getEffectiveTheme(theme)
  document.documentElement.classList.toggle('dark', effective === 'dark')
}

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>(() => {
    return (localStorage.getItem('slic-theme') as Theme) || 'system'
  })

  useEffect(() => {
    applyTheme(theme)
    if (theme === 'system') {
      localStorage.removeItem('slic-theme')
    } else {
      localStorage.setItem('slic-theme', theme)
    }
  }, [theme])

  // Listen for system theme changes when in system mode
  useEffect(() => {
    if (theme !== 'system') return
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => applyTheme('system')
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [theme])

  const icons: Record<Theme, string> = { light: '☀️', dark: '🌙', system: '💻' }
  const next: Record<Theme, Theme> = { system: 'light', light: 'dark', dark: 'system' }

  return (
    <button
      onClick={() => setTheme(next[theme])}
      className="px-2 py-1.5 border border-gray-300 dark:border-gray-600 rounded text-sm hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer"
      title={`Theme: ${theme}`}
    >
      {icons[theme]}
    </button>
  )
}
