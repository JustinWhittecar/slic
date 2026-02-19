import { useState, useEffect, useCallback, useRef } from 'react'
import { useAuth } from '../contexts/AuthContext'
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

const ENGINE_TYPES = ['Fusion', 'XL', 'XXL', 'Light', 'Compact', 'Primitive', 'ICE', 'Fuel Cell', 'Fission'] as const
const DEFAULT_ENGINES = ['Fusion', 'XL', 'XXL']
const HEAT_SINK_TYPES = ['All', 'Single', 'Double'] as const

const FILTER_KEYS: (keyof MechFilters)[] = [
  'name', 'tonnage_min', 'tonnage_max', 'era', 'tech_base', 'role',
  'bv_min', 'bv_max', 'tmm_min', 'armor_pct_min', 'heat_neutral_min', 'max_damage_min',
  'game_damage_min', 'combat_rating_min', 'combat_rating_max',
  'intro_year_min', 'intro_year_max', 'walk_mp_min', 'jump_mp_min',
  'engine_types', 'heat_sink_type',
]

interface FilterBarProps {
  filters: MechFilters
  onFiltersChange: (filters: MechFilters) => void
}

export function FilterBar({ filters, onFiltersChange }: FilterBarProps) {
  const { user } = useAuth()
  const [expanded, setExpanded] = useState(false)
  const [searchText, setSearchText] = useState(filters.name ?? '')
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()
  const initialized = useRef(false)

  const activeWeight = WEIGHT_CLASSES.find(
    w => w.min === filters.tonnage_min && w.max === filters.tonnage_max
  )?.label ?? 'All'

  const activeFilterCount = FILTER_KEYS.filter(k => {
    if (k === 'name') return false
    if (k === 'engine_types') {
      const et = filters.engine_types
      // Don't count as "active" if it's the default set
      if (!et || et.length === 0) return false
      if (et.length === DEFAULT_ENGINES.length && DEFAULT_ENGINES.every(e => et.includes(e))) return false
      return true
    }
    return filters[k] !== undefined
  }).length

  const selectedEngines = filters.engine_types ?? DEFAULT_ENGINES

  const toggleEngine = (eng: string) => {
    const current = [...selectedEngines]
    const idx = current.indexOf(eng)
    if (idx >= 0) {
      current.splice(idx, 1)
    } else {
      current.push(eng)
    }
    onFiltersChange({ ...filters, engine_types: current.length > 0 ? current : undefined })
  }

  const allEnginesSelected = selectedEngines.length === ENGINE_TYPES.length
  const toggleAllEngines = () => {
    if (allEnginesSelected) {
      onFiltersChange({ ...filters, engine_types: DEFAULT_ENGINES })
    } else {
      onFiltersChange({ ...filters, engine_types: [...ENGINE_TYPES] })
    }
  }

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

  // Load from URL on mount, apply defaults
  useEffect(() => {
    if (initialized.current) return
    initialized.current = true
    const params = new URLSearchParams(window.location.search)
    const f: MechFilters = {}
    const numKeys: (keyof MechFilters)[] = [
      'tonnage_min', 'tonnage_max', 'bv_min', 'bv_max', 'tmm_min',
      'armor_pct_min', 'heat_neutral_min', 'max_damage_min',
      'game_damage_min', 'combat_rating_min', 'combat_rating_max',
      'intro_year_min', 'intro_year_max', 'walk_mp_min', 'jump_mp_min',
    ]
    const strKeys: (keyof MechFilters)[] = ['name', 'era', 'tech_base', 'role', 'heat_sink_type']
    for (const k of numKeys) {
      const v = params.get(k)
      if (v) (f as any)[k] = Number(v)
    }
    for (const k of strKeys) {
      const v = params.get(k)
      if (v) (f as any)[k] = v
    }
    // Engine types from URL or default
    const et = params.get('engine_types')
    if (et) {
      f.engine_types = et.split(',').map(s => s.trim()).filter(Boolean)
    } else {
      f.engine_types = DEFAULT_ENGINES
    }
    setSearchText(f.name ?? '')
    onFiltersChange(f)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Sync to URL
  useEffect(() => {
    const params = new URLSearchParams()
    for (const k of FILTER_KEYS) {
      const v = filters[k]
      if (v === undefined || v === '') continue
      if (Array.isArray(v)) {
        if (v.length > 0) params.set(k, v.join(','))
      } else {
        params.set(k, String(v))
      }
    }
    const search = params.toString()
    const newUrl = search ? `?${search}` : window.location.pathname
    window.history.replaceState(null, '', newUrl)
  }, [filters])

  const numInput = (label: string, key: keyof MechFilters, placeholder: string, width = 'w-24') => (
    <div>
      <div className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>{label}</div>
      <input
        type="number"
        placeholder={placeholder}
        value={(filters[key] as number) ?? ''}
        onChange={e => {
          const val = e.target.value ? Number(e.target.value) : undefined
          onFiltersChange({ ...filters, [key]: val })
        }}
        className={`w-full sm:${width} px-2 py-1.5 text-xs rounded tabular-nums outline-none`}
        style={{
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border-default)',
          color: 'var(--text-primary)',
        }}
      />
    </div>
  )

  const pillButton = (label: string, isActive: boolean, onClick: () => void) => (
    <button
      onClick={onClick}
      className="px-2.5 py-1.5 text-xs rounded cursor-pointer transition-colors min-h-[36px] sm:min-h-0 sm:py-1"
      style={{
        background: isActive ? 'var(--accent)' : 'transparent',
        color: isActive ? '#ffffff' : 'var(--text-secondary)',
        border: isActive ? '1px solid var(--accent)' : '1px solid var(--border-default)',
      }}
    >
      {label}
    </button>
  )

  const selectInput = (label: string, key: keyof MechFilters, options: readonly string[]) => (
    <div>
      <div className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>{label}</div>
      <select
        value={filters[key] as string ?? ''}
        onChange={e => onFiltersChange({ ...filters, [key]: e.target.value || undefined })}
        className="px-2 py-1 text-xs rounded"
        style={{
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border-default)',
          color: 'var(--text-primary)',
        }}
      >
        <option value="">All</option>
        {options.filter(o => o !== 'All').map(o => (
          <option key={o} value={o}>{o}</option>
        ))}
      </select>
    </div>
  )

  const clearFilters = () => {
    setSearchText('')
    onFiltersChange({ engine_types: DEFAULT_ENGINES })
  }

  return (
    <div className="mb-4 space-y-3">
      <div className="flex gap-2 items-center">
        <input
          type="text"
          value={searchText}
          onChange={e => handleSearch(e.target.value)}
          placeholder="Search by name or model code (e.g. HBK-4P)..."
          className="flex-1 px-3 py-2 rounded text-sm outline-none"
          style={{
            background: 'var(--bg-surface)',
            border: '1px solid var(--border-default)',
            color: 'var(--text-primary)',
          }}
        />
        {user && (
          <button
            onClick={() => onFiltersChange({ ...filters, owned_only: !filters.owned_only })}
            className="px-3 py-2 rounded text-sm cursor-pointer whitespace-nowrap"
            style={{
              border: `1px solid ${filters.owned_only ? 'var(--accent)' : 'var(--border-default)'}`,
              color: filters.owned_only ? '#fff' : 'var(--text-secondary)',
              background: filters.owned_only ? 'var(--accent)' : 'var(--bg-surface)',
            }}
          >
            Owned
          </button>
        )}
        <button
          onClick={() => setExpanded(!expanded)}
          className="px-3 py-2 rounded text-sm cursor-pointer"
          style={{
            border: '1px solid var(--border-default)',
            color: activeFilterCount > 0 ? 'var(--accent)' : 'var(--text-secondary)',
            background: 'var(--bg-surface)',
          }}
        >
          Filters{activeFilterCount > 0 ? ` (${activeFilterCount})` : ''} {expanded ? '▲' : '▼'}
        </button>
      </div>

      {expanded && (
        <div className="p-3 rounded space-y-3" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)' }}>
          {/* Row 1: Weight, Tech Base */}
          <div className="flex flex-col sm:flex-row flex-wrap gap-4 items-start">
            <div>
              <div className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Weight Class</div>
              <div className="flex gap-1">
                {WEIGHT_CLASSES.map(w => {
                  const isActive = activeWeight === w.label
                  return pillButton(w.label, isActive, () => onFiltersChange({ ...filters, tonnage_min: w.min, tonnage_max: w.max }))
                })}
              </div>
            </div>

            <div>
              <div className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Tech Base</div>
              <div className="flex gap-1">
                {TECH_BASES.map(tb => {
                  const filterVal = tb === 'All' ? undefined : tb
                  const isActive = filters.tech_base === filterVal || (tb === 'All' && !filters.tech_base)
                  return pillButton(tb, isActive, () => onFiltersChange({ ...filters, tech_base: filterVal }))
                })}
              </div>
            </div>

            <div>
              <div className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Era</div>
              <select
                value={filters.era ?? ''}
                onChange={e => onFiltersChange({ ...filters, era: e.target.value || undefined })}
                className="px-2 py-1 text-xs rounded"
                style={{
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border-default)',
                  color: 'var(--text-primary)',
                }}
              >
                <option value="">All Eras</option>
                {ERAS.map(era => (
                  <option key={era} value={era}>{era}</option>
                ))}
              </select>
            </div>

            {selectInput('Heat Sinks', 'heat_sink_type', HEAT_SINK_TYPES)}
          </div>

          {/* Row 2: Engine types multi-select */}
          <div>
            <div className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Engine Type</div>
            <div className="flex gap-1 flex-wrap">
              {pillButton('All', allEnginesSelected, toggleAllEngines)}
              {ENGINE_TYPES.map(eng =>
                pillButton(eng, selectedEngines.includes(eng), () => toggleEngine(eng))
              )}
            </div>
          </div>

          {/* Row 3: Numeric range filters */}
          <div className="grid grid-cols-2 sm:flex sm:flex-wrap gap-4 items-start">
            {numInput('BV Min', 'bv_min', 'e.g. 1000')}
            {numInput('BV Max', 'bv_max', 'e.g. 2000')}
            {numInput('Year Min', 'intro_year_min', 'e.g. 2750')}
            {numInput('Year Max', 'intro_year_max', 'e.g. 3150')}
            {numInput('TMM ≥', 'tmm_min', 'any')}
            {numInput('Walk MP ≥', 'walk_mp_min', 'any')}
            {numInput('Jump MP ≥', 'jump_mp_min', 'any')}
          </div>

          {/* Row 4: Damage & rating filters */}
          <div className="grid grid-cols-2 sm:flex sm:flex-wrap gap-4 items-start">
            {numInput('HN Dmg ≥', 'heat_neutral_min', 'any')}
            {numInput('Alpha Dmg ≥', 'max_damage_min', 'any')}
            {numInput('Armor % ≥', 'armor_pct_min', 'any')}
            {numInput('Combat Rtg ≥', 'combat_rating_min', 'any')}
            {numInput('Combat Rtg ≤', 'combat_rating_max', 'any')}

            {activeFilterCount > 0 && (
              <div className="flex items-end">
                <button
                  onClick={clearFilters}
                  className="px-2 py-1 text-xs rounded cursor-pointer"
                  style={{ color: 'var(--text-tertiary)', border: '1px solid var(--border-default)' }}
                >
                  Clear All
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
