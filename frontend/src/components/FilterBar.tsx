import { useState, useEffect, useCallback, useRef } from 'react'
import type { MechFilters } from '../api/client'

const WEIGHT_CLASSES = [
  { label: 'All', min: undefined, max: undefined },
  { label: 'Light', min: 20, max: 35 },
  { label: 'Medium', min: 40, max: 55 },
  { label: 'Heavy', min: 60, max: 75 },
  { label: 'Assault', min: 80, max: 100 },
] as const

const TECH_BASES = ['All', 'Inner Sphere', 'Clan', 'Mixed'] as const

const ERAS = [
  'Age of War', 'Star League', 'Early Succession Wars', 'Late Succession Wars',
  'Clan Invasion', 'Civil War', 'Jihad', 'Dark Age', 'ilClan',
]

interface FilterBarProps {
  filters: MechFilters
  onFiltersChange: (filters: MechFilters) => void
}

export function FilterBar({ filters, onFiltersChange }: FilterBarProps) {
  const [expanded, setExpanded] = useState(false)
  const [searchText, setSearchText] = useState(filters.name ?? '')
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()

  const activeWeight = WEIGHT_CLASSES.find(
    w => w.min === filters.tonnage_min && w.max === filters.tonnage_max
  )?.label ?? 'All'

  const handleSearch = useCallback((value: string) => {
    setSearchText(value)
    clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      onFiltersChange({ ...filters, name: value || undefined })
    }, 300)
  }, [filters, onFiltersChange])

  useEffect(() => {
    return () => clearTimeout(debounceRef.current)
  }, [])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const f: MechFilters = {}
    if (params.get('name')) f.name = params.get('name')!
    if (params.get('tonnage_min')) f.tonnage_min = Number(params.get('tonnage_min'))
    if (params.get('tonnage_max')) f.tonnage_max = Number(params.get('tonnage_max'))
    if (params.get('era')) f.era = params.get('era')!
    if (params.get('tech_base')) f.tech_base = params.get('tech_base')!
    if (params.get('bv_min')) f.bv_min = Number(params.get('bv_min'))
    if (params.get('bv_max')) f.bv_max = Number(params.get('bv_max'))
    if (params.get('tmm_min')) f.tmm_min = Number(params.get('tmm_min'))
    if (params.get('armor_pct_min')) f.armor_pct_min = Number(params.get('armor_pct_min'))
    if (params.get('heat_neutral_min')) f.heat_neutral_min = Number(params.get('heat_neutral_min'))
    if (Object.keys(f).length > 0) {
      setSearchText(f.name ?? '')
      onFiltersChange(f)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    const params = new URLSearchParams()
    if (filters.name) params.set('name', filters.name)
    if (filters.tonnage_min) params.set('tonnage_min', String(filters.tonnage_min))
    if (filters.tonnage_max) params.set('tonnage_max', String(filters.tonnage_max))
    if (filters.era) params.set('era', filters.era)
    if (filters.tech_base) params.set('tech_base', filters.tech_base)
    if (filters.bv_min) params.set('bv_min', String(filters.bv_min))
    if (filters.bv_max) params.set('bv_max', String(filters.bv_max))
    if (filters.tmm_min) params.set('tmm_min', String(filters.tmm_min))
    if (filters.armor_pct_min) params.set('armor_pct_min', String(filters.armor_pct_min))
    if (filters.heat_neutral_min) params.set('heat_neutral_min', String(filters.heat_neutral_min))
    const search = params.toString()
    const newUrl = search ? `?${search}` : window.location.pathname
    window.history.replaceState(null, '', newUrl)
  }, [filters])

  const numInput = (label: string, key: keyof MechFilters, placeholder: string) => (
    <div>
      <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">{label}</div>
      <input
        type="number"
        placeholder={placeholder}
        value={filters[key] ?? ''}
        onChange={e => {
          const val = e.target.value ? Number(e.target.value) : undefined
          onFiltersChange({ ...filters, [key]: val })
        }}
        className="w-24 px-2 py-1 text-xs border border-gray-200 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 tabular-nums"
      />
    </div>
  )

  return (
    <div className="mb-4 space-y-3">
      <div className="flex gap-2 items-center">
        <input
          type="text"
          value={searchText}
          onChange={e => handleSearch(e.target.value)}
          placeholder="Search mechs..."
          className="flex-1 px-3 py-2 border border-gray-200 dark:border-gray-600 rounded text-sm outline-none focus:border-gray-400 dark:focus:border-gray-400 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500"
        />
        <button
          onClick={() => setExpanded(!expanded)}
          className="px-3 py-2 border border-gray-200 dark:border-gray-600 rounded text-sm text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer"
        >
          Filters {expanded ? '▲' : '▼'}
        </button>
      </div>

      {expanded && (
        <div className="flex flex-wrap gap-4 items-start p-3 border border-gray-200 dark:border-gray-600 rounded bg-gray-50 dark:bg-gray-800">
          {/* Weight Class */}
          <div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Weight Class</div>
            <div className="flex gap-1">
              {WEIGHT_CLASSES.map(w => (
                <button
                  key={w.label}
                  onClick={() => onFiltersChange({ ...filters, tonnage_min: w.min, tonnage_max: w.max })}
                  className={`px-2 py-1 text-xs rounded border cursor-pointer ${
                    activeWeight === w.label
                      ? 'bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 border-gray-900 dark:border-gray-100'
                      : 'border-gray-200 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                  }`}
                >
                  {w.label}
                </button>
              ))}
            </div>
          </div>

          {/* Tech Base */}
          <div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Tech Base</div>
            <div className="flex gap-1">
              {TECH_BASES.map(tb => {
                const filterVal = tb === 'All' ? undefined : tb
                const isActive = filters.tech_base === filterVal
                  || (tb === 'All' && !filters.tech_base)
                return (
                  <button
                    key={tb}
                    onClick={() => onFiltersChange({ ...filters, tech_base: filterVal })}
                    className={`px-2 py-1 text-xs rounded border cursor-pointer ${
                      isActive
                        ? 'bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 border-gray-900 dark:border-gray-100'
                        : 'border-gray-200 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                    }`}
                  >
                    {tb}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Era */}
          <div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Era</div>
            <select
              value={filters.era ?? ''}
              onChange={e => onFiltersChange({ ...filters, era: e.target.value || undefined })}
              className="px-2 py-1 text-xs border border-gray-200 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            >
              <option value="">All Eras</option>
              {ERAS.map(era => (
                <option key={era} value={era}>{era}</option>
              ))}
            </select>
          </div>

          {/* Stat Filters */}
          {numInput('BV Min', 'bv_min', 'e.g. 1000')}
          {numInput('BV Max', 'bv_max', 'e.g. 2000')}
          {numInput('TMM ≥', 'tmm_min', 'e.g. 2')}
          {numInput('Armor % ≥', 'armor_pct_min', 'e.g. 80')}
          {numInput('Heat Neutral Dmg ≥', 'heat_neutral_min', 'e.g. 30')}
          {numInput('Max Dmg ≥', 'max_damage_min', 'e.g. 40')}
        </div>
      )}
    </div>
  )
}
