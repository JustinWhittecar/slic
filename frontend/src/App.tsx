import { useState, useCallback } from 'react'
import { MechTable } from './components/MechTable'
import { FilterBar } from './components/FilterBar'
import { MechDetail } from './components/MechDetail'
import { CompareView } from './components/CompareView'
import { ThemeToggle } from './components/ThemeToggle'
import type { MechFilters } from './api/client'

export function App() {
  const [filters, setFilters] = useState<MechFilters>({})
  const [selectedMechId, setSelectedMechId] = useState<number | null>(null)
  const [compareIds, setCompareIds] = useState<number[]>([])
  const [showCompare, setShowCompare] = useState(false)
  const [count, setCount] = useState(0)

  const handleCountChange = useCallback((c: number) => setCount(c), [])

  const toggleCompare = useCallback((id: number) => {
    setCompareIds(prev => {
      if (prev.includes(id)) return prev.filter(x => x !== id)
      if (prev.length >= 4) return prev // max 4
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

  return (
    <div className="min-h-screen bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 transition-colors">
      <div className="max-w-[1400px] mx-auto px-4 py-4" style={{ fontFamily: 'system-ui, -apple-system, sans-serif' }}>
        <header className="mb-4 flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight">SLIC</h1>
            <p className="text-xs text-gray-500 dark:text-gray-400">BattleTech Mech Database</p>
          </div>
          <ThemeToggle />
        </header>

        <FilterBar filters={filters} onFiltersChange={setFilters} />

        {/* Compare bar */}
        {compareIds.length > 0 && (
          <div className="mb-2 flex items-center gap-2 bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800 rounded px-3 py-2">
            <span className="text-sm font-medium text-blue-700 dark:text-blue-300">
              {compareIds.length} mech{compareIds.length !== 1 ? 's' : ''} selected
            </span>
            {compareIds.length >= 2 && (
              <button
                onClick={() => setShowCompare(true)}
                className="text-sm bg-blue-600 text-white px-3 py-1 rounded hover:bg-blue-700 cursor-pointer"
              >
                Compare
              </button>
            )}
            <button
              onClick={() => { setCompareIds([]); setShowCompare(false) }}
              className="text-sm text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-200 ml-auto cursor-pointer"
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
        />

        <footer className="mt-2 text-xs text-gray-400 dark:text-gray-500">
          {count} variant{count !== 1 ? 's' : ''} shown
        </footer>

        {selectedMechId !== null && !showCompare && (
          <MechDetail mechId={selectedMechId} onClose={() => setSelectedMechId(null)} />
        )}

        {showCompare && compareIds.length >= 2 && (
          <CompareView
            mechIds={compareIds}
            onClose={() => setShowCompare(false)}
            onRemove={removeFromCompare}
          />
        )}
      </div>
    </div>
  )
}
