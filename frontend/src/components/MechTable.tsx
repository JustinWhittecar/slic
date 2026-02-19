import { useEffect, useState, useCallback, useMemo, useRef } from 'react'
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  createColumnHelper,
  flexRender,
  type SortingState,
  type VisibilityState,
} from '@tanstack/react-table'
import { useVirtualizer } from '@tanstack/react-virtual'
import { fetchMechs, fetchCollectionSummary, type MechListItem, type MechFilters } from '../api/client'
import { ColumnSelector } from './ColumnSelector'

const DEFAULT_VISIBILITY: VisibilityState = {
  name: true, tonnage: false, tech_base: true, bv: true, role: false, move: true, armor_total: false,
  heat_neutral_damage: true, alpha_damage: false, optimal_range: true, combat_rating: true, bv_efficiency: false, tmm: true, armor_coverage_pct: true,
  era: false, intro_year: true,
  engine_type: false, engine_rating: false, heat_sinks: false,
  rules_level: false, source: false, config: false,
}

const COLUMN_DEFS_META = [
  { id: 'name', label: 'Name' },
  { id: 'tonnage', label: 'Tonnage' },
  { id: 'tech_base', label: 'Tech Base' },
  { id: 'bv', label: 'BV' },
  { id: 'role', label: 'Role' },
  { id: 'era', label: 'Era' },
  { id: 'move', label: 'Move' },
  { id: 'armor_total', label: 'Armor Total' },
  { id: 'tmm', label: 'TMM' },
  { id: 'heat_neutral_damage', label: 'Heat Neutral Dmg' },
  { id: 'alpha_damage', label: 'Alpha Strike Dmg' },
  { id: 'optimal_range', label: 'Optimal Range' },
  { id: 'combat_rating', label: 'Combat Rating' },
  { id: 'bv_efficiency', label: 'BV Efficiency' },
  { id: 'armor_coverage_pct', label: 'Armor %' },
  { id: 'engine_type', label: 'Engine Type' },
  { id: 'engine_rating', label: 'Engine Rating' },
  { id: 'heat_sinks', label: 'Heat Sinks' },
  { id: 'rules_level', label: 'Rules Level' },
  { id: 'source', label: 'Source' },
  { id: 'config', label: 'Config' },
  { id: 'intro_year', label: 'Intro Year' },
]

function loadState<T>(key: string, fallback: T): T {
  try {
    const v = localStorage.getItem(key)
    return v ? JSON.parse(v) : fallback
  } catch { return fallback }
}

interface MechTableProps {
  filters: MechFilters
  onSelectMech: (id: number) => void
  selectedMechId: number | null
  onCountChange: (count: number) => void
  compareIds?: number[]
  onToggleCompare?: (id: number) => void
  onAddToList?: (mech: MechListItem) => void
  onClearFilters?: () => void
}

const columnHelper = createColumnHelper<MechListItem>()

export function MechTable({ filters, onSelectMech, selectedMechId, onCountChange, compareIds = [], onToggleCompare, onAddToList, onClearFilters }: MechTableProps) {
  const [mechs, setMechs] = useState<MechListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'intro_year', desc: false },
    { id: 'name', desc: false },
  ])
  const isMobile = typeof window !== 'undefined' && window.innerWidth < 640
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(
    () => {
      const saved = loadState('slic-columns', DEFAULT_VISIBILITY)
      if (isMobile) {
        // On mobile, hide more columns by default for a cleaner view
        return { ...saved, tech_base: false, optimal_range: false, armor_coverage_pct: false, tmm: false, intro_year: false }
      }
      return saved
    }
  )

  const parentRef = useRef<HTMLDivElement>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      // Strip owned_only from API filters (client-side filter)
      const { owned_only, ...apiFilters } = filters
      const data = await fetchMechs(apiFilters)

      if (owned_only) {
        try {
          const summary = await fetchCollectionSummary()
          const ownedChassisNames = new Set(summary.map(s => s.chassis_name))
          const filtered = data.filter(m => ownedChassisNames.has(m.chassis))
          setMechs(filtered)
          onCountChange(filtered.length)
        } catch {
          // If collection fetch fails, show all
          setMechs(data)
          onCountChange(data.length)
        }
      } else {
        setMechs(data)
        onCountChange(data.length)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [filters, onCountChange])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    localStorage.setItem('slic-columns', JSON.stringify(columnVisibility))
  }, [columnVisibility])

  const columns = useMemo(() => [
    ...(onToggleCompare ? [columnHelper.display({
      id: 'compare',
      header: () => <span className="text-xs tooltip-header" data-tip="Select 2-4 mechs to compare side by side"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{display:'inline'}}><path d="M12 3v18M3 12h18"/></svg></span>,
      cell: ({ row }) => {
        const checked = compareIds.includes(row.original.id)
        return (
          <input
            type="checkbox"
            checked={checked}
            onChange={(e) => { e.stopPropagation(); onToggleCompare(row.original.id) }}
            onClick={(e) => e.stopPropagation()}
            className="cursor-pointer"
            style={{ accentColor: 'var(--accent)' }}
          />
        )
      },
      size: 32,
      enableSorting: false,
    })] : []),
    ...(onAddToList ? [columnHelper.display({
      id: 'addToList',
      header: () => <span className="text-xs tooltip-header" data-tip="Add mech to your list builder">+</span>,
      cell: ({ row }) => (
        <button
          onClick={(e) => { e.stopPropagation(); onAddToList(row.original) }}
          className="text-xs cursor-pointer px-1 rounded"
          style={{ color: 'var(--accent)' }}
          onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-elevated)')}
          onMouseLeave={e => (e.currentTarget.style.background = '')}
          title="Add to list"
        >
          +
        </button>
      ),
      size: 32,
      enableSorting: false,
    })] : []),
    columnHelper.accessor(row => `${row.chassis} ${row.model_code}`, {
      id: 'name',
      header: 'Name',
      cell: ({ row }) => (
        <div>
          <span>{row.original.chassis} {row.original.model_code}</span>
          {row.original.alternate_name && (
            <span className="ml-1.5 text-xs italic" style={{ color: 'var(--text-tertiary)' }}>
              ({row.original.alternate_name})
            </span>
          )}
        </div>
      ),
    }),
    columnHelper.accessor('tonnage', {
      id: 'tonnage',
      header: 'Tons',
      cell: info => <span className="tabular-nums">{info.getValue()}</span>,
    }),
    columnHelper.accessor('tech_base', {
      id: 'tech_base',
      header: 'Tech',
    }),
    columnHelper.accessor('battle_value', {
      id: 'bv',
      header: 'BV',
      cell: info => <span className="tabular-nums">{info.getValue() ?? '—'}</span>,
    }),
    columnHelper.accessor('role', {
      id: 'role',
      header: 'Role',
      cell: info => info.getValue() || '—',
    }),
    columnHelper.accessor('era', {
      id: 'era',
      header: 'Era',
      cell: info => info.getValue() || '—',
    }),
    columnHelper.accessor(row => {
      const w = row.walk_mp ?? 0
      const r = row.run_mp ?? 0
      const j = row.jump_mp ?? 0
      return j > 0 ? `${w}/${r}/${j}` : `${w}/${r}`
    }, {
      id: 'move',
      header: 'Move',
      cell: info => <span className="tabular-nums">{info.getValue()}</span>,
      sortingFn: (a, b) => (a.original.walk_mp ?? 0) - (b.original.walk_mp ?? 0),
    }),
    columnHelper.accessor('armor_total', {
      id: 'armor_total',
      header: 'Armor',
      cell: info => <span className="tabular-nums">{info.getValue() ?? '—'}</span>,
    }),
    columnHelper.accessor('tmm', {
      id: 'tmm',
      header: () => <span className="tooltip-header" data-tip="Target Movement Modifier — penalty opponents take to hit this mech based on its speed">TMM</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null ? `+${info.getValue()}` : '—'}</span>,
    }),
    columnHelper.accessor('heat_neutral_damage', {
      id: 'heat_neutral_damage',
      header: () => <span className="tooltip-header" data-tip="Maximum damage output while staying heat-neutral (dissipating all heat generated). Uses optimal range band.">HN Damage</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null && info.getValue()! > 0 ? info.getValue()!.toFixed(1) : '—'}</span>,
    }),
    columnHelper.accessor('max_damage', {
      id: 'alpha_damage',
      header: () => <span className="tooltip-header" data-tip="Maximum possible damage firing all weapons simultaneously (alpha strike). Ignores heat.">Alpha Dmg</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null && info.getValue()! > 0 ? info.getValue()!.toFixed(1) : '—'}</span>,
    }),
    columnHelper.accessor('heat_neutral_range', {
      id: 'optimal_range',
      header: () => <span className="tooltip-header" data-tip="Optimal range in hexes — where this mech deals maximum heat-neutral damage">Optimal Range</span>,
      cell: info => {
        const v = info.getValue()
        return <span className="tabular-nums">{v && v !== '0' ? `${v} hex` : '—'}</span>
      },
      sortingFn: (a, b) => (parseInt(a.original.heat_neutral_range ?? '0') || 0) - (parseInt(b.original.heat_neutral_range ?? '0') || 0),
    }),
    columnHelper.accessor('combat_rating', {
      id: 'combat_rating',
      header: () => <span className="tooltip-header" data-tip="1-10 combat rating from 1,000 Monte Carlo simulations vs HBK-4P. Models damage spread, crits, ammo, heat, flanking, and physical attacks. 5 = HBK-4P baseline.">Combat Rating</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null && info.getValue()! > 0 ? info.getValue()!.toFixed(1) : '—'}</span>,
    }),
    columnHelper.accessor('bv_efficiency', {
      id: 'bv_efficiency',
      header: () => <span className="tooltip-header" data-tip="Combat Rating² per 1,000 BV. Rewards mechs that are both strong AND cheap. Higher = more combat value per BV spent.">BV Efficiency</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null ? info.getValue()!.toFixed(2) : '—'}</span>,
    }),
    columnHelper.accessor('armor_coverage_pct', {
      id: 'armor_coverage_pct',
      header: () => <span className="tooltip-header" data-tip="Percentage of maximum possible armor points allocated to this variant">Armor %</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null ? `${info.getValue()!.toFixed(1)}%` : '—'}</span>,
    }),
    columnHelper.accessor('engine_type', {
      id: 'engine_type',
      header: 'Engine Type',
      cell: info => info.getValue() ?? '—',
    }),
    columnHelper.accessor('engine_rating', {
      id: 'engine_rating',
      header: 'Engine',
      cell: info => <span className="tabular-nums">{info.getValue() ?? '—'}</span>,
    }),
    columnHelper.accessor('heat_sink_count', {
      id: 'heat_sinks',
      header: 'HS',
      cell: info => <span className="tabular-nums">{info.getValue() ?? '—'}</span>,
    }),
    columnHelper.accessor('rules_level', {
      id: 'rules_level',
      header: 'Rules',
      cell: info => info.getValue() ?? '—',
    }),
    columnHelper.accessor('source', {
      id: 'source',
      header: 'Source',
      cell: info => info.getValue() ?? '—',
    }),
    columnHelper.accessor('config', {
      id: 'config',
      header: 'Config',
      cell: info => info.getValue() ?? '—',
    }),
    columnHelper.accessor('intro_year', {
      id: 'intro_year',
      header: 'Year',
      cell: info => <span className="tabular-nums">{info.getValue() ?? '—'}</span>,
    }),
  ], [compareIds, onToggleCompare, onAddToList])

  const table = useReactTable({
    data: mechs,
    columns,
    state: { sorting, columnVisibility },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    enableMultiSort: true,
  })

  const { rows } = table.getRowModel()

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 36,
    overscan: 20,
  })

  // Reset scroll when filters change
  useEffect(() => {
    if (parentRef.current) parentRef.current.scrollTop = 0
  }, [filters])

  if (error) return (
    <div className="flex flex-col items-center justify-center p-12 gap-3" style={{ color: 'var(--text-secondary)' }}>
      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ color: 'var(--text-tertiary)' }}>
        <circle cx="12" cy="12" r="10"/><path d="M12 8v4m0 4h.01"/>
      </svg>
      <div className="text-sm font-medium">Something went wrong</div>
      <div className="text-xs" style={{ color: 'var(--text-tertiary)' }}>{error}</div>
      <button onClick={load} className="text-xs px-4 py-2 rounded cursor-pointer font-medium" style={{ background: 'var(--accent)', color: '#fff' }}>
        Retry
      </button>
    </div>
  )

  return (
    <div>
      <div className="flex justify-end mb-2">
        <ColumnSelector
          columns={COLUMN_DEFS_META}
          visibility={columnVisibility}
          onVisibilityChange={setColumnVisibility}
          defaultVisibility={DEFAULT_VISIBILITY}
        />
      </div>
      <div
        ref={parentRef}
        className="rounded overflow-auto"
        style={{ height: 'calc(100vh - 200px)', border: '1px solid var(--border-default)' }}
      >
        {loading ? (
          <div className="p-0">
            {Array.from({ length: 15 }).map((_, i) => (
              <div key={i} className="flex gap-3 px-3 py-2" style={{ borderBottom: '1px solid var(--border-subtle)' }}>
                <div className="h-4 rounded animate-pulse" style={{ width: '30%', background: 'var(--bg-elevated)' }} />
                <div className="h-4 rounded animate-pulse" style={{ width: '12%', background: 'var(--bg-elevated)' }} />
                <div className="h-4 rounded animate-pulse" style={{ width: '10%', background: 'var(--bg-elevated)' }} />
                <div className="h-4 rounded animate-pulse" style={{ width: '15%', background: 'var(--bg-elevated)' }} />
                <div className="h-4 rounded animate-pulse" style={{ width: '10%', background: 'var(--bg-elevated)' }} />
              </div>
            ))}
          </div>
        ) : (
          <table className="w-full text-sm fade-in" style={{ fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif' }}>
            <thead className="sticky top-0 z-10" style={{ background: 'var(--bg-surface)' }}>
              {table.getHeaderGroups().map(hg => (
                <tr key={hg.id} style={{ borderBottom: '1px solid var(--border-default)' }}>
                  {hg.headers.map(header => (
                    <th
                      key={header.id}
                      onClick={header.column.getToggleSortingHandler()}
                      className="px-3 py-2 text-left text-xs font-medium uppercase tracking-wide cursor-pointer select-none"
                      style={{ color: 'var(--text-tertiary)' }}
                      onMouseEnter={e => (e.currentTarget.style.color = 'var(--text-primary)')}
                      onMouseLeave={e => (e.currentTarget.style.color = 'var(--text-tertiary)')}
                    >
                      {flexRender(header.column.columnDef.header, header.getContext())}
                      {{ asc: ' ↑', desc: ' ↓' }[header.column.getIsSorted() as string] ?? ''}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {virtualizer.getVirtualItems().length === 0 && (
                <tr>
                  <td colSpan={columns.length} className="p-12 text-center">
                    <div className="flex flex-col items-center gap-3">
                      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" style={{ color: 'var(--text-tertiary)', opacity: 0.5 }}>
                        <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
                      </svg>
                      <div className="text-sm font-medium" style={{ color: 'var(--text-secondary)' }}>No mechs match your filters</div>
                      <div className="text-xs" style={{ color: 'var(--text-tertiary)' }}>Try broadening your search or adjusting filter criteria</div>
                      {onClearFilters && (
                        <button onClick={onClearFilters} className="text-xs px-4 py-2 rounded cursor-pointer font-medium mt-1" style={{ background: 'var(--accent)', color: '#fff' }}>
                          Clear Filters
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              )}
              {virtualizer.getVirtualItems().length > 0 && (
                <>
                  {virtualizer.getVirtualItems()[0].start > 0 && (
                    <tr><td style={{ height: virtualizer.getVirtualItems()[0].start }} /></tr>
                  )}
                  {virtualizer.getVirtualItems().map(virtualRow => {
                    const row = rows[virtualRow.index]
                    const isSelected = row.original.id === selectedMechId
                    const isComparing = compareIds.includes(row.original.id)
                    return (
                      <tr
                        key={row.id}
                        onClick={() => onSelectMech(row.original.id)}
                        className="cursor-pointer transition-colors"
                        style={{
                          height: 36,
                          borderBottom: '1px solid var(--border-subtle)',
                          borderLeft: isSelected ? '3px solid var(--accent)' : '3px solid transparent',
                          color: 'var(--text-primary)',
                          background: isSelected
                            ? 'var(--bg-elevated)'
                            : isComparing
                            ? 'var(--bg-surface)'
                            : undefined,
                        }}
                        onMouseEnter={e => { if (!isSelected) { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.borderLeft = '3px solid var(--accent)' } }}
                        onMouseLeave={e => { if (!isSelected) { e.currentTarget.style.borderLeft = '3px solid transparent'; if (!isComparing) e.currentTarget.style.background = ''; else e.currentTarget.style.background = 'var(--bg-surface)' } }}
                      >
                        {row.getVisibleCells().map(cell => (
                          <td key={cell.id} className="px-3 py-1.5 whitespace-nowrap">
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </td>
                        ))}
                      </tr>
                    )
                  })}
                  {virtualizer.getVirtualItems().length > 0 && (
                    <tr>
                      <td style={{
                        height: virtualizer.getTotalSize() -
                          (virtualizer.getVirtualItems()[virtualizer.getVirtualItems().length - 1]?.end ?? 0)
                      }} />
                    </tr>
                  )}
                </>
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
