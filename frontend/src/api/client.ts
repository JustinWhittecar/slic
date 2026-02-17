const BASE = '/api'

export interface MechListItem {
  id: number
  model_code: string
  name: string
  chassis: string
  tonnage: number
  tech_base: string
  battle_value?: number
  intro_year?: number
  era?: string
  role?: string
  walk_mp?: number
  run_mp?: number
  jump_mp?: number
  armor_total?: number
  engine_type?: string
  engine_rating?: number
  heat_sink_count?: number
  heat_sink_type?: string
  rules_level?: string
  source?: string
  config?: string
  tmm?: number
  armor_coverage_pct?: number
  heat_neutral_damage?: number
  max_damage?: number
  effective_heat_neutral_damage?: number
  heat_neutral_range?: string
  game_damage?: number
}

export interface MechEquipment {
  id: number
  name: string
  type: string
  location: string
  quantity: number
  damage?: number
  heat?: number
  tonnage: number
  slots: number
  internal_name?: string
  bv?: number
  rack_size?: number
  expected_damage?: number
  damage_per_ton?: number
  damage_per_heat?: number
  to_hit_modifier?: number
  min_range?: number
  short_range?: number
  medium_range?: number
  long_range?: number
  effective_damage_short?: number
  effective_damage_medium?: number
  effective_damage_long?: number
  effective_dps_ton?: number
  effective_dps_heat?: number
}

export interface MechDetail extends MechListItem {
  sarna_url?: string
  stats?: {
    walk_mp: number
    run_mp: number
    jump_mp: number
    armor_total: number
    internal_structure_total: number
    heat_sink_count: number
    heat_sink_type: string
    engine_type: string
    engine_rating: number
    cockpit_type?: string
    gyro_type?: string
    myomer_type?: string
    structure_type?: string
    armor_type?: string
    tmm?: number
    armor_coverage_pct?: number
    heat_neutral_damage?: number
    heat_neutral_range?: string
    max_damage?: number
    effective_heat_neutral_damage?: number
  }
  equipment?: MechEquipment[]
}

export interface MechFilters {
  name?: string
  tonnage_min?: number
  tonnage_max?: number
  era?: string
  faction?: string
  role?: string
  tech_base?: string
  bv_min?: number
  bv_max?: number
  tmm_min?: number
  armor_pct_min?: number
  heat_neutral_min?: number
  max_damage_min?: number
}

export async function fetchMechs(filters: MechFilters = {}): Promise<MechListItem[]> {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== '') {
      params.set(key, String(value))
    }
  }
  const url = `${BASE}/mechs${params.toString() ? '?' + params : ''}`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`Failed to fetch mechs: ${res.status}`)
  return res.json()
}

export async function fetchMech(id: number): Promise<MechDetail> {
  const res = await fetch(`${BASE}/mechs/${id}`)
  if (!res.ok) throw new Error(`Failed to fetch mech: ${res.status}`)
  return res.json()
}
