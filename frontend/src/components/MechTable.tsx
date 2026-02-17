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
import { fetchMechs, type MechListItem, type MechFilters } from '../api/client'
import { ColumnSelector } from './ColumnSelector'

const DEFAULT_VISIBILITY: VisibilityState = {
  name: true, tonnage: false, tech_base: true, bv: true, role: false, move: true, armor_total: false,
  game_damage: true, tmm: true, armor_coverage_pct: true,
  era: false, intro_year: true,
  engine_type: false, engine_rating: false, heat_sinks: false, run_mp: false,
  rules_level: false, source: false, config: false, intro_year: false,
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
  { id: 'game_damage', label: 'Game Damage' },
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
}

const columnHelper = createColumnHelper<MechListItem>()

export function MechTable({ filters, onSelectMech, selectedMechId, onCountChange }: MechTableProps) {
  const [mechs, setMechs] = useState<MechListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'intro_year', desc: false },
    { id: 'name', desc: false },
  ])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(
    () => loadState('slic-columns', DEFAULT_VISIBILITY)
  )

  const parentRef = useRef<HTMLDivElement>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchMechs(filters)
      setMechs(data)
      onCountChange(data.length)
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
    columnHelper.accessor(row => `${row.chassis} ${row.model_code}`, {
      id: 'name',
      header: 'Name',
      cell: info => info.getValue(),
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
    columnHelper.accessor('game_damage', {
      id: 'game_damage',
      header: () => <span className="tooltip-header" data-tip="Expected total damage over a 12-turn game on 2 mapsheets. Simulates both mechs walking toward each other (ref opponent: 4/6 mover). Accounts for range brackets, min range penalties, 2d6 hit probability (Gunnery 4, walked, TMM +2), and heat-neutral weapon selection each turn.">Game Dmg</span>,
      cell: info => <span className="tabular-nums">{info.getValue() != null && info.getValue()! > 0 ? info.getValue()!.toFixed(1) : '—'}</span>,
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
  ], [])

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

  if (error) return <div className="text-red-600 dark:text-red-400 text-sm p-4">Error: {error}</div>

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
        className="border border-gray-200 dark:border-gray-700 rounded overflow-auto"
        style={{ height: 'calc(100vh - 240px)' }}
      >
        {loading ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400 text-sm">Loading...</div>
        ) : (
          <table className="w-full text-sm" style={{ fontFamily: 'system-ui, -apple-system, sans-serif' }}>
            <thead className="sticky top-0 bg-white dark:bg-gray-900 z-10">
              {table.getHeaderGroups().map(hg => (
                <tr key={hg.id} className="border-b-2 border-gray-200 dark:border-gray-700">
                  {hg.headers.map(header => (
                    <th
                      key={header.id}
                      onClick={header.column.getToggleSortingHandler()}
                      className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide cursor-pointer select-none whitespace-nowrap hover:text-gray-900 dark:hover:text-gray-100"
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
                  <td colSpan={columns.length} className="p-8 text-center text-gray-400 dark:text-gray-500">
                    No mechs found.
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
                    return (
                      <tr
                        key={row.id}
                        onClick={() => onSelectMech(row.original.id)}
                        className={`border-b border-gray-100 dark:border-gray-800 cursor-pointer transition-colors ${
                          isSelected
                            ? 'bg-blue-50 dark:bg-gray-700'
                            : 'hover:bg-gray-50 dark:hover:bg-gray-800'
                        }`}
                        style={{ height: 36 }}
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
