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
import { fetchMechs, fetchCollectionSummary, fetchPreferences, savePreferences, deletePreferences, type MechListItem, type MechFilters } from '../api/client'
import { ColumnSelector } from './ColumnSelector'
import { useAuth } from '../contexts/AuthContext'

export const DEFAULT_COLUMN_ORDER = [
  'name', 'tonnage', 'tech_base', 'role', 'bv', 'move', 'tmm', 'combat_rating', 'bv_efficiency',
  'goonhammer', 'era', 'intro_year', 'armor_total', 'heat_neutral_damage', 'alpha_damage', 'optimal_range',
  'armor_coverage_pct', 'engine_type', 'engine_rating', 'heat_sinks', 'rules_level', 'source', 'config',
]

export const DEFAULT_VISIBILITY: VisibilityState = {
  name: true, tonnage: true, tech_base: true, bv: true, role: true, move: true,
  tmm: true, combat_rating: true, bv_efficiency: true,
  // Hidden by default
  armor_total: false, heat_neutral_damage: false, alpha_damage: false, optimal_range: false,
  armor_coverage_pct: false, era: false, intro_year: false,
  engine_type: false, engine_rating: false, heat_sinks: false,
  goonhammer: false,
  rules_level: false, source: false, config: false,
}

export const COLUMN_DEFS_META = [
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
  { id: 'goonhammer', label: 'Goonhammer' },
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
  const { user } = useAuth()
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
        return { ...saved, tech_base: false, optimal_range: false, armor_coverage_pct: false, tmm: false, intro_year: false }
      }
      return saved
    }
  )
  const [columnOrder, setColumnOrder] = useState<string[]>(
    () => loadState('slic-column-order', DEFAULT_COLUMN_ORDER)
  )

  // Drag state
  const [dragCol, setDragCol] = useState<string | null>(null)
  const [dragOver, setDragOver] = useState<string | null>(null)

  // Debounced preferences save
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const prefsLoadedRef = useRef(false)

  const parentRef = useRef<HTMLDivElement>(null)

  // Load preferences from API on login
  useEffect(() => {
    if (!user || prefsLoadedRef.current) return
    prefsLoadedRef.current = true
    fetchPreferences().then(prefs => {
      if (prefs.column_visibility) {
        const cv = typeof prefs.column_visibility === 'string'
          ? JSON.parse(prefs.column_visibility as any) : prefs.column_visibility
        setColumnVisibility(cv)
      }
      if (prefs.column_order) {
        const co = typeof prefs.column_order === 'string'
          ? JSON.parse(prefs.column_order as any) : prefs.column_order
        if (Array.isArray(co) && co.length > 0) setColumnOrder(co)
      }
    }).catch(() => {})
  }, [user])

  // Save preferences (debounced)
  const debouncedSave = useCallback((vis: VisibilityState, order: string[]) => {
    clearTimeout(saveTimerRef.current)
    // Always save to localStorage
    localStorage.setItem('slic-columns', JSON.stringify(vis))
    localStorage.setItem('slic-column-order', JSON.stringify(order))

    if (user) {
      saveTimerRef.current = setTimeout(() => {
        savePreferences({ column_visibility: vis, column_order: order }).catch(() => {})
      }, 500)
    }
  }, [user])

  useEffect(() => {
    debouncedSave(columnVisibility, columnOrder)
  }, [columnVisibility, columnOrder, debouncedSave])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
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

  const resetToDefaults = useCallback(() => {
    setColumnVisibility(DEFAULT_VISIBILITY)
    setColumnOrder(DEFAULT_COLUMN_ORDER)
    if (user) {
      deletePreferences().catch(() => {})
    }
    localStorage.removeItem('slic-columns')
    localStorage.removeItem('slic-column-order')
  }, [user])

  const columns = useMemo(() => [
    ...(onToggleCompare ? [columnHelper.display({
      id: 'compare',
      header: () => <span className="text-xs tooltip-header" data-tip="Compare"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{display:'inline'}}><path d="M12 3v18M3 12h18"/></svg></span>,
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
      header: () => <span className="text-xs tooltip-header" data-tip="Add to list">+</span>,
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
          <button
            className="mech-name-link"
            onClick={(e) => { e.stopPropagation(); onSelectMech(row.original.id) }}
            style={{ background: 'none', border: 'none', padding: 0, cursor: 'pointer', color: 'var(--accent)', textDecoration: 'none', fontWeight: 500, textAlign: 'left' }}
          >
            {row.original.chassis} {row.original.model_code}
          </button>
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
      header: () => <span className="tooltip-header" data-tip="Battle Value (4/5 pilot)">BV</span>,
      cell: info => <span className="tabular-nums">{info.getValue()?.toLocaleString() ?? '—'}</span>,
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
      header: () => <span className="tooltip-header" data-tip="Target Movement Modifier">TMM</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null ? `+${info.getValue()}` : '—'}</span>,
    }),
    columnHelper.accessor('heat_neutral_damage', {
      id: 'heat_neutral_damage',
      header: () => <span className="tooltip-header" data-tip="Heat-neutral damage at optimal range">HN Dmg</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null && info.getValue()! > 0 ? info.getValue()!.toFixed(1) : '—'}</span>,
    }),
    columnHelper.accessor('max_damage', {
      id: 'alpha_damage',
      header: () => <span className="tooltip-header" data-tip="Alpha strike damage (all weapons, ignores heat)">Alpha</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null && info.getValue()! > 0 ? info.getValue()!.toFixed(1) : '—'}</span>,
    }),
    columnHelper.accessor('heat_neutral_range', {
      id: 'optimal_range',
      header: () => <span className="tooltip-header" data-tip="Best range band in hexes">Range</span>,
      cell: info => {
        const v = info.getValue()
        return <span className="tabular-nums">{v && v !== '0' ? `${v} hex` : '—'}</span>
      },
      sortingFn: (a, b) => (parseInt(a.original.heat_neutral_range ?? '0') || 0) - (parseInt(b.original.heat_neutral_range ?? '0') || 0),
    }),
    columnHelper.accessor('combat_rating', {
      id: 'combat_rating',
      header: () => <span className="tooltip-header" data-tip="Monte Carlo sim score, 1–10 (HBK-4P = 5)">CR</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null && info.getValue()! > 0 ? info.getValue()!.toFixed(1) : '—'}</span>,
    }),
    columnHelper.accessor('bv_efficiency', {
      id: 'bv_efficiency',
      header: () => <span className="tooltip-header" data-tip="Combat value per BV spent (HBK-4P = 5)">BV Eff</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null ? info.getValue()!.toFixed(1) : '—'}</span>,
    }),
    columnHelper.accessor('armor_coverage_pct', {
      id: 'armor_coverage_pct',
      header: () => <span className="tooltip-header" data-tip="Armor as % of max possible">Armor %</span>,
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
    columnHelper.accessor('goonhammer_rating', {
      id: 'goonhammer',
      header: () => <span className="tooltip-header" data-tip="Goonhammer expert letter grade (F–S)">GH</span>,
      cell: info => {
        const v = info.getValue()
        if (!v) return <span style={{ color: 'var(--text-tertiary)' }}>—</span>
        const g = v.replace(/[+-]/g, '').toUpperCase()
        const colors: Record<string, string> = { S: '#e040fb', A: '#4caf50', B: '#2196f3', C: '#ff9800', D: '#f44336', F: '#9e9e9e' }
        const color = colors[g] || 'var(--text-secondary)'
        return <span className="tabular-nums font-semibold" style={{ color }}>{v}</span>
      },
      sortingFn: (a, b) => {
        const order = ['F-','F','F+','D-','D','D+','C-','C','C+','B-','B','B+','A-','A','A+','S']
        const ai = order.indexOf(a.original.goonhammer_rating ?? '')
        const bi = order.indexOf(b.original.goonhammer_rating ?? '')
        return (ai === -1 ? -1 : ai) - (bi === -1 ? -1 : bi)
      },
    }),
  ], [compareIds, onToggleCompare, onAddToList])

  const table = useReactTable({
    data: mechs,
    columns,
    state: { sorting, columnVisibility, columnOrder },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    onColumnOrderChange: setColumnOrder,
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

  useEffect(() => {
    if (parentRef.current) parentRef.current.scrollTop = 0
  }, [filters])

  // Drag handlers
  const handleDragStart = (colId: string) => {
    if (colId === 'name' || colId === 'compare' || colId === 'addToList') return
    setDragCol(colId)
  }
  const handleDragOver = (e: React.DragEvent, colId: string) => {
    e.preventDefault()
    if (colId === 'name' || colId === 'compare' || colId === 'addToList') return
    setDragOver(colId)
  }
  const handleDrop = (targetColId: string) => {
    if (!dragCol || dragCol === targetColId) {
      setDragCol(null)
      setDragOver(null)
      return
    }
    if (targetColId === 'name' || targetColId === 'compare' || targetColId === 'addToList') {
      setDragCol(null)
      setDragOver(null)
      return
    }
    setColumnOrder(prev => {
      const order = [...prev]
      const fromIdx = order.indexOf(dragCol)
      const toIdx = order.indexOf(targetColId)
      if (fromIdx === -1 || toIdx === -1) return prev
      order.splice(fromIdx, 1)
      order.splice(toIdx, 0, dragCol)
      return order
    })
    setDragCol(null)
    setDragOver(null)
  }

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
      <div className="flex justify-end mb-2 gap-2">
        <button
          onClick={resetToDefaults}
          className="px-3 py-2 rounded text-sm cursor-pointer"
          style={{ border: '1px solid var(--border-default)', color: 'var(--text-tertiary)' }}
        >
          Reset Columns
        </button>
        <ColumnSelector
          columns={COLUMN_DEFS_META}
          visibility={columnVisibility}
          onVisibilityChange={setColumnVisibility}
          defaultVisibility={DEFAULT_VISIBILITY}
          columnOrder={columnOrder}
          onColumnOrderChange={setColumnOrder}
        />
      </div>
      <div
        ref={parentRef}
        className="rounded overflow-auto"
        style={{ height: 'calc(100vh - 200px)', border: '1px solid var(--border-default)', WebkitOverflowScrolling: 'touch' }}
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
          <table className="text-sm fade-in" style={{ fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif', minWidth: '100%' }}>
            <thead className="sticky top-0 z-10" style={{ background: 'var(--bg-surface)' }}>
              {table.getHeaderGroups().map(hg => (
                <tr key={hg.id} style={{ borderBottom: '1px solid var(--border-default)' }}>
                  {hg.headers.map(header => {
                    const colId = header.column.id
                    const isDraggable = colId !== 'name' && colId !== 'compare' && colId !== 'addToList'
                    const isDragTarget = dragOver === colId
                    return (
                      <th
                        key={header.id}
                        draggable={isDraggable}
                        onDragStart={() => handleDragStart(colId)}
                        onDragOver={e => handleDragOver(e, colId)}
                        onDragEnd={() => { setDragCol(null); setDragOver(null) }}
                        onDrop={() => handleDrop(colId)}
                        className="px-3 py-2 text-left text-xs font-medium uppercase tracking-wide select-none whitespace-nowrap"
                        style={{
                          color: 'var(--text-tertiary)',
                          cursor: isDraggable ? 'grab' : 'default',
                          borderLeft: isDragTarget ? '2px solid var(--accent)' : '2px solid transparent',
                          opacity: dragCol === colId ? 0.4 : 1,
                          ...(colId === 'name' ? { position: 'sticky' as const, left: 0, zIndex: 20, background: 'var(--bg-surface)' } : {}),
                        }}
                        onMouseEnter={e => (e.currentTarget.style.color = 'var(--text-primary)')}
                        onMouseLeave={e => (e.currentTarget.style.color = 'var(--text-tertiary)')}
                      >
                        <span
                          onClick={header.column.getToggleSortingHandler()}
                          className="cursor-pointer"
                        >
                          {isDraggable && <span className="mr-1" style={{ color: 'var(--text-tertiary)', fontSize: '10px' }}>⋮⋮</span>}
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {{ asc: ' ↑', desc: ' ↓' }[header.column.getIsSorted() as string] ?? ''}
                        </span>
                      </th>
                    )
                  })}
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
                          <td
                            key={cell.id}
                            className="px-3 py-1.5 whitespace-nowrap"
                            style={cell.column.id === 'name' ? { position: 'sticky', left: 0, zIndex: 1, background: 'inherit' } : undefined}
                          >
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
