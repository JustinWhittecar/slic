const BASE = '/api'

const fetchWithCreds = (url: string, opts?: RequestInit) =>
  fetch(url, { credentials: 'include', ...opts })

export interface AuthUser {
  id: number
  email: string
  display_name: string
  avatar_url: string
}

export async function fetchMe(): Promise<AuthUser | null> {
  try {
    const res = await fetchWithCreds(`${BASE}/auth/me`)
    if (!res.ok) return null
    return res.json()
  } catch {
    return null
  }
}

export async function logout(): Promise<void> {
  await fetchWithCreds(`${BASE}/auth/logout`, { method: 'POST' })
}

export interface MechListItem {
  id: number
  model_code: string
  name: string
  chassis: string
  alternate_name?: string
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
  combat_rating?: number
  bv_efficiency?: number
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
  iwm_url?: string
  catalyst_url?: string
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
    has_targeting_computer?: boolean
    combat_rating?: number
    offense_turns?: number
    defense_turns?: number
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
  game_damage_min?: number
  combat_rating_min?: number
  combat_rating_max?: number
  intro_year_min?: number
  intro_year_max?: number
  walk_mp_min?: number
  jump_mp_min?: number
  engine_types?: string[]
  heat_sink_type?: string
  owned_only?: boolean
}

export async function fetchMechs(filters: MechFilters = {}): Promise<MechListItem[]> {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== '') {
      if (Array.isArray(value)) {
        if (value.length > 0) params.set(key, value.join(','))
      } else {
        params.set(key, String(value))
      }
    }
  }
  const url = `${BASE}/mechs${params.toString() ? '?' + params : ''}`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`Failed to fetch mechs: ${res.status}`)
  const mechs: MechListItem[] = await res.json()
  for (const m of mechs) {
    if (m.combat_rating && m.combat_rating > 0 && m.battle_value && m.battle_value > 0) {
      m.bv_efficiency = (m.combat_rating * m.combat_rating) / (m.battle_value / 1000)
    }
  }
  return mechs
}

export async function fetchMech(id: number): Promise<MechDetail> {
  const res = await fetch(`${BASE}/mechs/${id}`)
  if (!res.ok) throw new Error(`Failed to fetch mech: ${res.status}`)
  return res.json()
}

// Physical Models
export interface PhysicalModel {
  id: number
  name: string
  manufacturer: string
  sku?: string
  source_url?: string
  image_url?: string
  in_print: boolean
}

export interface ChassisModels {
  chassis_id: number
  chassis_name: string
  tonnage: number
  tech_base: string
  has_model: boolean
  models: PhysicalModel[]
}

export async function fetchModels(chassisId?: number, includeProxy?: boolean): Promise<ChassisModels[]> {
  const searchParams = new URLSearchParams()
  if (chassisId) searchParams.set('chassis_id', String(chassisId))
  if (includeProxy) searchParams.set('include_proxy', 'true')
  const qs = searchParams.toString()
  const res = await fetch(`${BASE}/models${qs ? '?' + qs : ''}`)
  if (!res.ok) throw new Error(`Failed to fetch models: ${res.status}`)
  return res.json()
}

// Collection
export interface CollectionItem {
  id: number
  physical_model_id: number
  quantity: number
  notes: string
  model_name: string
  manufacturer: string
  sku: string
  source_url?: string
  chassis_id: number
  chassis_name: string
  tonnage: number
}

export interface CollectionSummaryItem {
  chassis_id: number
  chassis_name: string
  total_quantity: number
}

export async function fetchCollection(): Promise<CollectionItem[]> {
  const res = await fetchWithCreds(`${BASE}/collection`)
  if (!res.ok) throw new Error(`Failed to fetch collection: ${res.status}`)
  return res.json()
}

export async function updateCollection(modelId: number, quantity: number, notes?: string): Promise<void> {
  await fetchWithCreds(`${BASE}/collection/${modelId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ quantity, notes: notes ?? '' }),
  })
}

export async function deleteFromCollection(modelId: number): Promise<void> {
  await fetchWithCreds(`${BASE}/collection/${modelId}`, { method: 'DELETE' })
}

export async function fetchCollectionSummary(): Promise<CollectionSummaryItem[]> {
  const res = await fetchWithCreds(`${BASE}/collection/summary`)
  if (!res.ok) throw new Error(`Failed to fetch collection summary: ${res.status}`)
  return res.json()
}
