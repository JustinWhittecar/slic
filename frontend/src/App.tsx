import { useState, useCallback } from 'react'
import { MechTable } from './components/MechTable'
import { FilterBar } from './components/FilterBar'
import { MechDetail } from './components/MechDetail'
import { ThemeToggle } from './components/ThemeToggle'
import type { MechFilters } from './api/client'

export function App() {
  const [filters, setFilters] = useState<MechFilters>({})
  const [selectedMechId, setSelectedMechId] = useState<number | null>(null)
  const [count, setCount] = useState(0)

  const handleCountChange = useCallback((c: number) => setCount(c), [])

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
        <MechTable
          filters={filters}
          onSelectMech={setSelectedMechId}
          selectedMechId={selectedMechId}
          onCountChange={handleCountChange}
        />

        <footer className="mt-2 text-xs text-gray-400 dark:text-gray-500">
          {count} variant{count !== 1 ? 's' : ''} shown
        </footer>

        {selectedMechId !== null && (
          <MechDetail mechId={selectedMechId} onClose={() => setSelectedMechId(null)} />
        )}
      </div>
    </div>
  )
}
