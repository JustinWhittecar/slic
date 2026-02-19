import { useEffect, useRef, useState } from 'react'

interface ChangelogPageProps {
  onClose: () => void
}

interface ChangelogEntry {
  date: string
  title: string
  items: string[]
}

const entries: ChangelogEntry[] = [
  {
    date: 'Feb 18, 2026',
    title: 'Launch',
    items: [
      '4,227 BattleMech variants with Combat Rating scores',
      'Monte Carlo combat simulation (1,000 runs per variant on 2D hex grids)',
      'BV Efficiency scoring (log-scale, anchored to HBK-4P = 5)',
      'Google OAuth login with collection tracking',
      '1,846 physical miniature models (IWM, Catalyst, Ral Partha, WizKids, FASA, Armorcast)',
      'Dynamic filter chips with addable filters',
      'Draggable column reorder with preference persistence',
      'List builder with BV budget tracking and pilot skill multipliers',
      'Mobile responsive design',
      'Feedback form (no GitHub account required)',
    ],
  },
]

export function ChangelogPage({ onClose }: ChangelogPageProps) {
  const [visible, setVisible] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
    return () => setVisible(false)
  }, [])

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) onClose()
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [onClose])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  return (
    <div className="fixed inset-0 z-50" style={{ background: 'rgba(0,0,0,0.4)' }}>
      <div
        ref={panelRef}
        className="absolute inset-0 sm:inset-auto sm:right-0 sm:top-0 sm:h-full sm:w-[520px] sm:max-w-full shadow-2xl overflow-y-auto transition-transform duration-200 ease-out"
        style={{
          transform: visible ? 'translateX(0)' : 'translateX(100%)',
          background: 'var(--bg-page)',
          borderLeft: '1px solid var(--border-default)',
        }}
      >
        <div className="flex flex-col">
          <div className="p-5 pb-3 flex justify-between items-start" style={{ borderBottom: '1px solid var(--border-default)' }}>
            <div>
              <h2 className="text-xl font-bold" style={{ color: 'var(--text-primary)' }}>Changelog</h2>
              <p className="text-sm mt-0.5" style={{ color: 'var(--text-secondary)' }}>What's new in SLIC</p>
            </div>
            <button
              onClick={onClose}
              className="text-lg cursor-pointer min-w-[44px] min-h-[44px] flex items-center justify-center"
              style={{ color: 'var(--text-tertiary)' }}
            >
              ✕
            </button>
          </div>

          <div className="p-5 space-y-6">
            {entries.map((entry, i) => (
              <div key={i}>
                <div className="flex items-baseline gap-2 mb-2">
                  <h3 className="text-sm font-bold uppercase tracking-wide" style={{ color: 'var(--text-primary)' }}>
                    {entry.title}
                  </h3>
                  <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>{entry.date}</span>
                </div>
                <ul className="list-none space-y-1 text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
                  {entry.items.map((item, j) => (
                    <li key={j} className="flex gap-2">
                      <span style={{ color: 'var(--accent)' }}>•</span>
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
