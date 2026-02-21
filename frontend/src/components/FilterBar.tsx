import { useState, useEffect, useCallback, useRef } from 'react'
import { useAuth } from '../contexts/AuthContext'
import type { MechFilters } from '../api/client'
import { fetchEquipmentNames } from '../api/client'
import type { EquipmentName } from '../api/client'
import { track } from '../analytics'

const WEIGHT_CLASSES = [
  { label: 'Light', min: 20, max: 35 },
  { label: 'Medium', min: 40, max: 55 },
  { label: 'Heavy', min: 60, max: 75 },
  { label: 'Assault', min: 80, max: 100 },
] as const

const TECH_BASES = ['Inner Sphere', 'Clan', 'Mixed'] as const
const ERAS = [
  'Age of War', 'Star League', 'Early Succession Wars', 'Late Succession Wars',
  'Clan Invasion', 'Civil War', 'Jihad', 'Dark Age', 'ilClan',
]
const ROLES = [
  'Juggernaut', 'Brawler', 'Skirmisher', 'Striker', 'Scout',
  'Missile Boat', 'Sniper', 'Ambusher', 'undefined',
]
const ENGINE_TYPES = ['Fusion', 'XL', 'XXL', 'Light', 'Compact', 'Primitive', 'ICE', 'Fuel Cell', 'Fission'] as const
const DEFAULT_ENGINES = ['Fusion', 'XL', 'XXL']
const HEAT_SINK_TYPES = ['Single', 'Double'] as const
const ARMOR_TYPES = [
  'Standard', 'Ferro-Fibrous', 'Light Ferro-Fibrous', 'Heavy Ferro-Fibrous',
  'Stealth', 'Reactive', 'Reflective', 'Hardened', 'Patchwork',
  'Ferro-Lamellor', 'Ballistic-Reinforced', 'Heat-Dissipating',
  'Anti-Penetrative Ablation', 'Impact-Resistant', 'Primitive',
  'Industrial', 'Heavy Industrial', 'Commercial',
] as const
const STRUCTURE_TYPES = [
  'Standard', 'Endo Steel', 'Endo-Composite', 'Composite',
  'Reinforced', 'Industrial', 'Endo Steel Prototype',
] as const

// Filter definitions
type FilterType = 'range' | 'enum' | 'multi-select' | 'equipment'
interface FilterDef {
  field: string
  label: string
  type: FilterType
  group: string
  options?: readonly string[]
  minKey?: keyof MechFilters
  maxKey?: keyof MechFilters
  filterKey?: keyof MechFilters
  placeholder?: string
}

const FILTER_DEFS: FilterDef[] = [
  // Identity
  { field: 'tech_base', label: 'Tech Base', type: 'enum', group: 'Identity', options: TECH_BASES, filterKey: 'tech_base' },
  { field: 'era', label: 'Era', type: 'enum', group: 'Identity', options: ERAS, filterKey: 'era' },
  { field: 'role', label: 'Role', type: 'enum', group: 'Identity', options: ROLES, filterKey: 'role' },
  { field: 'intro_year', label: 'Intro Year', type: 'range', group: 'Identity', minKey: 'intro_year_min', maxKey: 'intro_year_max', placeholder: 'e.g. 2750' },
  // Combat
  { field: 'bv', label: 'BV', type: 'range', group: 'Combat', minKey: 'bv_min', maxKey: 'bv_max', placeholder: 'e.g. 1500' },
  { field: 'tmm', label: 'TMM', type: 'range', group: 'Combat', minKey: 'tmm_min', placeholder: 'e.g. 3' },
  { field: 'combat_rating', label: 'Combat Rating', type: 'range', group: 'Combat', minKey: 'combat_rating_min', maxKey: 'combat_rating_max', placeholder: 'e.g. 5' },
  { field: 'bv_efficiency', label: 'BV Efficiency', type: 'range', group: 'Combat', minKey: 'combat_rating_min', maxKey: 'combat_rating_max' },
  { field: 'armor_pct', label: 'Armor %', type: 'range', group: 'Combat', minKey: 'armor_pct_min', placeholder: 'e.g. 80' },
  // Damage
  { field: 'heat_neutral', label: 'Heat Neutral Dmg', type: 'range', group: 'Damage', minKey: 'heat_neutral_min', placeholder: 'e.g. 20' },
  { field: 'alpha_damage', label: 'Alpha Dmg', type: 'range', group: 'Damage', minKey: 'max_damage_min', placeholder: 'e.g. 30' },
  // Technical
  { field: 'engine_types', label: 'Engine Type', type: 'multi-select', group: 'Technical', options: ENGINE_TYPES, filterKey: 'engine_types' },
  { field: 'heat_sink_type', label: 'Heat Sink Type', type: 'enum', group: 'Technical', options: HEAT_SINK_TYPES, filterKey: 'heat_sink_type' },
  { field: 'armor_type', label: 'Armor Type', type: 'enum', group: 'Technical', options: ARMOR_TYPES, filterKey: 'armor_type' },
  { field: 'structure_type', label: 'Structure Type', type: 'enum', group: 'Technical', options: STRUCTURE_TYPES, filterKey: 'structure_type' },
  { field: 'walk_mp', label: 'Walk MP', type: 'range', group: 'Technical', minKey: 'walk_mp_min', placeholder: 'e.g. 4' },
  { field: 'jump_mp', label: 'Jump MP', type: 'range', group: 'Technical', minKey: 'jump_mp_min', placeholder: 'e.g. 3' },
  // Equipment
  { field: 'equipment', label: 'Equipment', type: 'equipment', group: 'Equipment', filterKey: 'equipment' },
]

// What an active chip looks like
export interface ActiveFilterChip {
  field: string
  op: string
  value: string | string[]
}

interface FilterBarProps {
  filters: MechFilters
  onFiltersChange: (filters: MechFilters) => void
  activeChips?: ActiveFilterChip[]
  onActiveChipsChange?: (chips: ActiveFilterChip[]) => void
}

const FILTER_KEYS: (keyof MechFilters)[] = [
  'name', 'tonnage_min', 'tonnage_max', 'era', 'tech_base', 'role',
  'bv_min', 'bv_max', 'tmm_min', 'armor_pct_min', 'heat_neutral_min', 'max_damage_min',
  'game_damage_min', 'combat_rating_min', 'combat_rating_max',
  'intro_year_min', 'intro_year_max', 'walk_mp_min', 'jump_mp_min',
  'engine_types', 'heat_sink_type', 'armor_type', 'structure_type', 'equipment',
]

export function FilterBar({ filters, onFiltersChange }: FilterBarProps) {
  const { user } = useAuth()
  const [searchText, setSearchText] = useState(filters.name ?? '')
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()
  const initialized = useRef(false)
  const [showFilterMenu, setShowFilterMenu] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  // Derive active chips from filters
  const activeChips = deriveChipsFromFilters(filters)

  const hasActiveFilters = activeChips.length > 0 ||
    filters.tonnage_min !== undefined || filters.tonnage_max !== undefined ||
    filters.owned_only

  // Close menu on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setShowFilterMenu(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const handleSearch = useCallback((value: string) => {
    setSearchText(value)
    clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      onFiltersChange({ ...filters, name: value || undefined })
      if (value) track('search', { query: value })
    }, 300)
  }, [filters, onFiltersChange])

  useEffect(() => {
    return () => clearTimeout(debounceRef.current)
  }, [])

  // Load from URL on mount
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
    const strKeys: (keyof MechFilters)[] = ['name', 'era', 'tech_base', 'role', 'heat_sink_type', 'armor_type', 'structure_type']
    for (const k of numKeys) {
      const v = params.get(k)
      if (v) (f as any)[k] = Number(v)
    }
    for (const k of strKeys) {
      const v = params.get(k)
      if (v) (f as any)[k] = v
    }
    const et = params.get('engine_types')
    if (et) {
      f.engine_types = et.split(',').map(s => s.trim()).filter(Boolean)
    } else {
      f.engine_types = DEFAULT_ENGINES
    }
    const eq = params.get('equipment')
    if (eq) {
      f.equipment = eq.split('|').map(s => s.trim()).filter(Boolean)
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
        if (v.length > 0) params.set(k, v.join(k === 'equipment' ? '|' : ','))
      } else {
        params.set(k, String(v))
      }
    }
    const search = params.toString()
    const newUrl = search ? `?${search}` : window.location.pathname
    window.history.replaceState(null, '', newUrl)
  }, [filters])

  const activeWeight = WEIGHT_CLASSES.find(
    w => w.min === filters.tonnage_min && w.max === filters.tonnage_max
  )?.label ?? null

  const addFilter = (def: FilterDef) => {
    setShowFilterMenu(false)
    track('filter_add', { field: def.field })
    if (def.type === 'enum') {
      const firstOpt = def.options?.[0]
      if (firstOpt && def.filterKey) {
        onFiltersChange({ ...filters, [def.filterKey]: firstOpt })
      }
    } else if (def.type === 'multi-select') {
      // Already has engine_types default, don't change
    } else if (def.type === 'equipment') {
      onFiltersChange({ ...filters, equipment: filters.equipment ?? [] })
    } else if (def.type === 'range') {
      if (def.minKey) {
        onFiltersChange({ ...filters, [def.minKey]: 0 })
      }
    }
  }

  const removeFilter = (def: FilterDef) => {
    track('filter_remove', { field: def.field })
    const newFilters = { ...filters }
    if (def.type === 'enum' && def.filterKey) {
      delete (newFilters as any)[def.filterKey]
    } else if (def.type === 'equipment') {
      delete (newFilters as any).equipment
    } else if (def.type === 'multi-select' && def.filterKey) {
      if (def.field === 'engine_types') {
        newFilters.engine_types = DEFAULT_ENGINES
      } else {
        delete (newFilters as any)[def.filterKey]
      }
    } else if (def.type === 'range') {
      if (def.minKey) delete (newFilters as any)[def.minKey]
      if (def.maxKey) delete (newFilters as any)[def.maxKey]
    }
    onFiltersChange(newFilters)
  }

  const isFilterActive = (def: FilterDef): boolean => {
    if (def.type === 'enum' && def.filterKey) {
      return filters[def.filterKey] !== undefined
    }
    if (def.type === 'equipment') {
      return filters.equipment !== undefined
    }
    if (def.type === 'multi-select') {
      if (def.field === 'engine_types') {
        const et = filters.engine_types
        if (!et) return false
        return !(et.length === DEFAULT_ENGINES.length && DEFAULT_ENGINES.every(e => et.includes(e)))
      }
      return false
    }
    if (def.type === 'range') {
      return (def.minKey && filters[def.minKey] !== undefined) || (def.maxKey && filters[def.maxKey] !== undefined) || false
    }
    return false
  }

  const clearFilters = () => {
    setSearchText('')
    onFiltersChange({ engine_types: DEFAULT_ENGINES })
  }

  // Group filter defs for menu
  const groups = ['Identity', 'Combat', 'Damage', 'Technical', 'Equipment']

  // Render active chips
  const renderActiveChips = () => {
    const nodes: React.ReactNode[] = []
    for (const chip of activeChips) {
      const def = FILTER_DEFS.find(d => d.field === chip.field)
      if (!def) continue
      if (def.type === 'equipment') {
        nodes.push(
          <EquipmentFilterChip
            key="equipment"
            equipment={filters.equipment ?? []}
            onEquipmentChange={(eq) => onFiltersChange({ ...filters, equipment: eq.length > 0 ? eq : [] })}
            onRemove={() => removeFilter(def)}
          />
        )
      } else {
        nodes.push(
          <FilterChip
            key={chip.field + chip.op}
            chip={chip}
            def={def}
            filters={filters}
            onFiltersChange={onFiltersChange}
            onRemove={() => removeFilter(def)}
          />
        )
      }
    }
    return nodes
  }

  return (
    <div className="mb-4 space-y-2">
      {/* Row 1: Search + Weight pills + controls */}
      <div className="flex gap-2 items-center flex-wrap">
        <div className="relative flex-1 min-w-[200px]">
          <svg className="absolute left-2.5 top-1/2 -translate-y-1/2" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ color: 'var(--text-tertiary)' }}>
            <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
          </svg>
          <input
            type="text"
            value={searchText}
            onChange={e => handleSearch(e.target.value)}
            placeholder="Search by name or model code..."
            className="w-full pl-8 pr-3 py-2 rounded text-sm outline-none"
            style={{
              background: 'var(--bg-surface)',
              border: '1px solid var(--border-default)',
              color: 'var(--text-primary)',
            }}
          />
        </div>

        <div className="flex gap-1">
          {WEIGHT_CLASSES.map(w => (
            <button
              key={w.label}
              onClick={() => {
                if (activeWeight === w.label) {
                  onFiltersChange({ ...filters, tonnage_min: undefined, tonnage_max: undefined })
                } else {
                  onFiltersChange({ ...filters, tonnage_min: w.min, tonnage_max: w.max })
                }
              }}
              className="px-2.5 py-1.5 text-xs rounded cursor-pointer transition-colors"
              style={{
                background: activeWeight === w.label ? 'var(--accent)' : 'transparent',
                color: activeWeight === w.label ? '#ffffff' : 'var(--text-secondary)',
                border: activeWeight === w.label ? '1px solid var(--accent)' : '1px solid var(--border-default)',
              }}
            >
              {w.label}
            </button>
          ))}
        </div>

        {/* + Filter button */}
        <div className="relative" ref={menuRef}>
          <button
            onClick={() => setShowFilterMenu(!showFilterMenu)}
            className="px-2.5 py-1.5 text-xs rounded cursor-pointer flex items-center gap-1"
            style={{
              background: 'var(--bg-surface)',
              color: 'var(--text-secondary)',
              border: '1px solid var(--border-default)',
            }}
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 5v14M5 12h14"/></svg>
            Filter
          </button>

          {showFilterMenu && (
            <div className="absolute left-0 top-full mt-1 rounded shadow-lg z-50 w-56 py-1 max-h-80 overflow-y-auto"
              style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)' }}>
              {groups.map(group => (
                <div key={group}>
                  <div className="px-3 py-1 text-[10px] font-semibold uppercase tracking-wider" style={{ color: 'var(--text-tertiary)' }}>
                    {group}
                  </div>
                  {FILTER_DEFS.filter(d => d.group === group).map(def => {
                    const active = isFilterActive(def)
                    return (
                      <button
                        key={def.field}
                        onClick={() => !active && addFilter(def)}
                        disabled={active}
                        className="w-full text-left px-3 py-1.5 text-sm cursor-pointer flex items-center justify-between"
                        style={{
                          color: active ? 'var(--text-tertiary)' : 'var(--text-primary)',
                          opacity: active ? 0.5 : 1,
                        }}
                        onMouseEnter={e => { if (!active) e.currentTarget.style.background = 'var(--bg-hover)' }}
                        onMouseLeave={e => { e.currentTarget.style.background = '' }}
                      >
                        <span>{def.label}</span>
                        {active && <span className="text-[10px]">✓</span>}
                      </button>
                    )
                  })}
                </div>
              ))}
            </div>
          )}
        </div>

        {user && (
          <button
            onClick={() => onFiltersChange({ ...filters, owned_only: !filters.owned_only })}
            className="px-2.5 py-1.5 text-xs rounded cursor-pointer"
            style={{
              border: `1px solid ${filters.owned_only ? 'var(--accent)' : 'var(--border-default)'}`,
              color: filters.owned_only ? '#fff' : 'var(--text-secondary)',
              background: filters.owned_only ? 'var(--accent)' : 'transparent',
            }}
          >
            Owned
          </button>
        )}

        {hasActiveFilters && (
          <button
            onClick={clearFilters}
            className="px-2.5 py-1.5 text-xs rounded cursor-pointer"
            style={{ color: 'var(--text-tertiary)', border: '1px solid var(--border-default)' }}
          >
            Reset
          </button>
        )}
      </div>

      {/* Row 2: Active filter chips */}
      {activeChips.length > 0 && (
        <div className="flex gap-1.5 flex-wrap">
          {renderActiveChips()}
        </div>
      )}
    </div>
  )
}

// Derive chips from the current MechFilters state
function deriveChipsFromFilters(filters: MechFilters): ActiveFilterChip[] {
  const chips: ActiveFilterChip[] = []

  for (const def of FILTER_DEFS) {
    if (def.type === 'equipment') {
      if (filters.equipment !== undefined) {
        chips.push({ field: def.field, op: 'has', value: filters.equipment })
      }
      continue
    }
    if (def.type === 'enum' && def.filterKey) {
      const val = filters[def.filterKey]
      if (val !== undefined) {
        chips.push({ field: def.field, op: ':', value: String(val) })
      }
    } else if (def.type === 'multi-select') {
      if (def.field === 'engine_types') {
        const et = filters.engine_types
        if (et && !(et.length === DEFAULT_ENGINES.length && DEFAULT_ENGINES.every(e => et.includes(e)))) {
          chips.push({ field: def.field, op: 'in', value: et })
        }
      }
    } else if (def.type === 'range') {
      if (def.field === 'bv_efficiency') continue
      const minVal = def.minKey ? filters[def.minKey] : undefined
      const maxVal = def.maxKey ? filters[def.maxKey] : undefined
      if (minVal !== undefined || maxVal !== undefined) {
        if (minVal !== undefined && maxVal !== undefined) {
          chips.push({ field: def.field, op: 'between', value: `${minVal}–${maxVal}` })
        } else if (minVal !== undefined) {
          chips.push({ field: def.field, op: '≥', value: String(minVal) })
        } else if (maxVal !== undefined) {
          chips.push({ field: def.field, op: '≤', value: String(maxVal) })
        }
      }
    }
  }

  return chips
}

interface FilterChipProps {
  chip: ActiveFilterChip
  def: FilterDef
  filters: MechFilters
  onFiltersChange: (f: MechFilters) => void
  onRemove: () => void
}

function FilterChip({ chip, def, filters, onFiltersChange, onRemove }: FilterChipProps) {
  if (def.type === 'enum') {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs"
        style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-primary)' }}>
        <span style={{ color: 'var(--text-tertiary)' }}>{def.label}:</span>
        <select
          value={String(chip.value)}
          onChange={e => def.filterKey && onFiltersChange({ ...filters, [def.filterKey]: e.target.value || undefined })}
          className="bg-transparent text-xs outline-none cursor-pointer"
          style={{ color: 'var(--text-primary)' }}
        >
          {def.options?.map(o => <option key={o} value={o}>{o}</option>)}
        </select>
        <button onClick={onRemove} className="ml-0.5 cursor-pointer hover:opacity-70" style={{ color: 'var(--text-tertiary)' }}>×</button>
      </span>
    )
  }

  if (def.type === 'multi-select' && def.field === 'engine_types') {
    const selected = (filters.engine_types ?? DEFAULT_ENGINES)
    return (
      <span className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs flex-wrap"
        style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-primary)' }}>
        <span style={{ color: 'var(--text-tertiary)' }}>Engine:</span>
        {ENGINE_TYPES.map(eng => {
          const isOn = selected.includes(eng)
          return (
            <button
              key={eng}
              onClick={() => {
                const next = isOn ? selected.filter(e => e !== eng) : [...selected, eng]
                onFiltersChange({ ...filters, engine_types: next.length > 0 ? next : undefined })
              }}
              className="px-1.5 py-0.5 rounded text-[10px] cursor-pointer"
              style={{
                background: isOn ? 'var(--accent)' : 'transparent',
                color: isOn ? '#fff' : 'var(--text-tertiary)',
                border: isOn ? '1px solid var(--accent)' : '1px solid var(--border-subtle)',
              }}
            >
              {eng}
            </button>
          )
        })}
        <button onClick={onRemove} className="ml-0.5 cursor-pointer hover:opacity-70" style={{ color: 'var(--text-tertiary)' }}>×</button>
      </span>
    )
  }

  if (def.type === 'range') {
    const minVal = def.minKey ? (filters[def.minKey] as number | undefined) : undefined
    const maxVal = def.maxKey ? (filters[def.maxKey] as number | undefined) : undefined

    return (
      <span className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs"
        style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-primary)' }}>
        <span style={{ color: 'var(--text-tertiary)' }}>{def.label}</span>
        {def.minKey && (
          <>
            <span style={{ color: 'var(--text-tertiary)' }}>≥</span>
            <input
              type="number"
              value={minVal ?? ''}
              onChange={e => {
                const v = e.target.value ? Number(e.target.value) : undefined
                onFiltersChange({ ...filters, [def.minKey!]: v })
              }}
              placeholder={def.placeholder ?? ''}
              className="w-16 bg-transparent text-xs outline-none tabular-nums"
              style={{ color: 'var(--text-primary)' }}
            />
          </>
        )}
        {def.maxKey && (
          <>
            <span style={{ color: 'var(--text-tertiary)' }}>≤</span>
            <input
              type="number"
              value={maxVal ?? ''}
              onChange={e => {
                const v = e.target.value ? Number(e.target.value) : undefined
                onFiltersChange({ ...filters, [def.maxKey!]: v })
              }}
              placeholder={def.placeholder ?? ''}
              className="w-16 bg-transparent text-xs outline-none tabular-nums"
              style={{ color: 'var(--text-primary)' }}
            />
          </>
        )}
        <button onClick={onRemove} className="ml-0.5 cursor-pointer hover:opacity-70" style={{ color: 'var(--text-tertiary)' }}>×</button>
      </span>
    )
  }

  return null
}

interface EquipmentFilterChipProps {
  equipment: string[]
  onEquipmentChange: (eq: string[]) => void
  onRemove: () => void
}

function EquipmentFilterChip({ equipment, onEquipmentChange, onRemove }: EquipmentFilterChipProps) {
  const [search, setSearch] = useState('')
  const [suggestions, setSuggestions] = useState<EquipmentName[]>([])
  const [showDropdown, setShowDropdown] = useState(false)
  const [allEquipment, setAllEquipment] = useState<EquipmentName[]>([])
  const inputRef = useRef<HTMLInputElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // Load all equipment names once
  useEffect(() => {
    fetchEquipmentNames().then(setAllEquipment).catch(() => {})
  }, [])

  // Filter suggestions locally
  useEffect(() => {
    if (!search.trim()) {
      setSuggestions([])
      return
    }
    const q = search.toLowerCase()
    const filtered = allEquipment
      .filter(e => e.name.toLowerCase().includes(q) && !equipment.includes(e.name))
      .slice(0, 20)
    setSuggestions(filtered)
  }, [search, allEquipment, equipment])

  // Close dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const addEquipment = (name: string) => {
    if (!equipment.includes(name)) {
      onEquipmentChange([...equipment, name])
      track('filter_equipment_add', { name })
    }
    setSearch('')
    setShowDropdown(false)
    inputRef.current?.focus()
  }

  const removeEquipment = (name: string) => {
    onEquipmentChange(equipment.filter(e => e !== name))
  }

  return (
    <span className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs flex-wrap relative"
      style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-primary)' }}
      ref={dropdownRef}>
      <span style={{ color: 'var(--text-tertiary)' }}>Equipment:</span>
      {equipment.map(name => (
        <span key={name} className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px]"
          style={{ background: 'var(--accent)', color: '#fff' }}>
          {name}
          <button onClick={() => removeEquipment(name)} className="cursor-pointer hover:opacity-70">×</button>
        </span>
      ))}
      <div className="relative">
        <input
          ref={inputRef}
          type="text"
          value={search}
          onChange={e => { setSearch(e.target.value); setShowDropdown(true) }}
          onFocus={() => setShowDropdown(true)}
          placeholder="Search..."
          className="w-24 bg-transparent text-xs outline-none"
          style={{ color: 'var(--text-primary)' }}
        />
        {showDropdown && suggestions.length > 0 && (
          <div className="absolute left-0 top-full mt-1 rounded shadow-lg z-50 w-64 max-h-48 overflow-y-auto py-1"
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)' }}>
            {suggestions.map(s => (
              <button
                key={s.id}
                onClick={() => addEquipment(s.name)}
                className="w-full text-left px-3 py-1.5 text-xs cursor-pointer flex items-center justify-between"
                style={{ color: 'var(--text-primary)' }}
                onMouseEnter={e => e.currentTarget.style.background = 'var(--bg-hover)'}
                onMouseLeave={e => e.currentTarget.style.background = ''}
              >
                <span>{s.name}</span>
                <span className="text-[10px]" style={{ color: 'var(--text-tertiary)' }}>{s.type}</span>
              </button>
            ))}
          </div>
        )}
      </div>
      <button onClick={onRemove} className="ml-0.5 cursor-pointer hover:opacity-70" style={{ color: 'var(--text-tertiary)' }}>×</button>
    </span>
  )
}
