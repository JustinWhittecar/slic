import { useState, useCallback, useEffect, useRef } from 'react'
import { MechTable } from './components/MechTable'
import { FilterBar } from './components/FilterBar'
import { MechDetail } from './components/MechDetail'
import { CompareView } from './components/CompareView'
import { ThemeToggle } from './components/ThemeToggle'
import { ListBuilder } from './components/ListBuilder'
import { AboutPage } from './components/AboutPage'
import type { ListMech } from './components/ListBuilder'
import { fetchMechs, type MechListItem, type MechFilters } from './api/client'

let nextEntryId = 1

export function App() {
  const [filters, setFilters] = useState<MechFilters>({ engine_types: ['Fusion', 'XL', 'XXL'] })
  const [selectedMechId, setSelectedMechId] = useState<number | null>(null)
  const [compareIds, setCompareIds] = useState<number[]>([])
  const [showCompare, setShowCompare] = useState(false)
  const [count, setCount] = useState(0)
  const [totalCount, setTotalCount] = useState(0)
  const totalFetched = useRef(false)

  useEffect(() => {
    if (!totalFetched.current) {
      totalFetched.current = true
      fetchMechs({}).then(data => setTotalCount(data.length)).catch(() => {})
    }
  }, [])
  const [listMechs, setListMechs] = useState<ListMech[]>([])
  const [showListBuilder, setShowListBuilder] = useState(true)
  const [showAbout, setShowAbout] = useState(false)

  const handleCountChange = useCallback((c: number) => setCount(c), [])

  const toggleCompare = useCallback((id: number) => {
    setCompareIds(prev => {
      if (prev.includes(id)) return prev.filter(x => x !== id)
      if (prev.length >= 4) return prev
      return [...prev, id]
    })
  }, [])

  const removeFromCompare = useCallback((id: number) => {
    setCompareIds(prev => {
      const next = prev.filter(x => x !== id)
      if (next.length < 2) setShowCompare(false)
      return next
    })
  }, [])

  const addToList = useCallback((mech: MechListItem) => {
    setListMechs(prev => [...prev, {
      id: `entry-${nextEntryId++}`,
      mechData: mech,
      pilotGunnery: 4,
      pilotPiloting: 5,
    }])
    setShowListBuilder(true)
  }, [])

  // Keyboard shortcut: L to toggle list builder
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'l' || e.key === 'L') {
        const tag = (e.target as HTMLElement)?.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
        e.preventDefault()
        setShowListBuilder(s => !s)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  return (
    <div className="min-h-screen transition-colors" style={{ background: 'var(--bg-page)', color: 'var(--text-primary)', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif' }}>
      <div className="max-w-[1600px] mx-auto px-4 py-4">
        <header className="mb-4 flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight">SLIC</h1>
            <p className="text-xs" style={{ color: 'var(--text-tertiary)' }}>BattleTech Mech Database & List Builder</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowListBuilder(s => !s)}
              className="text-xs px-3 py-1.5 rounded cursor-pointer flex items-center gap-1.5"
              style={{
                background: showListBuilder ? 'var(--accent)' : 'var(--bg-elevated)',
                color: showListBuilder ? '#fff' : 'var(--text-secondary)',
                border: `1px solid ${showListBuilder ? 'var(--accent)' : 'var(--border-default)'}`,
              }}
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2"/><rect x="9" y="3" width="6" height="4" rx="1"/></svg>
              List{listMechs.length > 0 ? ` (${listMechs.length})` : ''}
            </button>
            <button
              onClick={() => setShowAbout(true)}
              className="text-xs px-3 py-1.5 rounded cursor-pointer"
              style={{
                background: 'var(--bg-elevated)',
                color: 'var(--text-secondary)',
                border: '1px solid var(--border-default)',
              }}
            >
              About
            </button>
            <ThemeToggle />
          </div>
        </header>

        <FilterBar filters={filters} onFiltersChange={setFilters} />

        {/* List Builder - inline */}
        {showListBuilder && (
          <ListBuilder
            mechs={listMechs}
            onMechsChange={setListMechs}
            onClose={() => setShowListBuilder(false)}
          />
        )}

        {compareIds.length > 0 && (
          <div className="mb-2 flex items-center gap-2 rounded px-3 py-2" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)' }}>
            <span className="text-sm font-medium" style={{ color: 'var(--accent)' }}>
              {compareIds.length} mech{compareIds.length !== 1 ? 's' : ''} selected
            </span>
            {compareIds.length >= 2 && (
              <button
                onClick={() => setShowCompare(true)}
                className="text-sm text-white px-3 py-1 rounded cursor-pointer"
                style={{ background: 'var(--accent)' }}
              >
                Compare
              </button>
            )}
            <button
              onClick={() => { setCompareIds([]); setShowCompare(false) }}
              className="text-sm ml-auto cursor-pointer"
              style={{ color: 'var(--text-secondary)' }}
            >
              Clear
            </button>
          </div>
        )}

        <MechTable
          filters={filters}
          onSelectMech={setSelectedMechId}
          selectedMechId={selectedMechId}
          onCountChange={handleCountChange}
          compareIds={compareIds}
          onToggleCompare={toggleCompare}
          onAddToList={addToList}
        />

        <footer className="mt-2 text-xs" style={{ color: 'var(--text-tertiary)' }}>
          Showing {count} of {totalCount} variants
        </footer>
      </div>

      {selectedMechId !== null && !showCompare && (
        <MechDetail mechId={selectedMechId} onClose={() => setSelectedMechId(null)} onAddToList={addToList} />
      )}

      {showCompare && compareIds.length >= 2 && (
        <CompareView
          mechIds={compareIds}
          onClose={() => setShowCompare(false)}
          onRemove={removeFromCompare}
        />
      )}

      {showAbout && <AboutPage onClose={() => setShowAbout(false)} />}
    </div>
  )
}
