import { useEffect, useState } from 'react'
import { fetchMech, type MechDetail, type MechListItem } from '../api/client'

interface CompareViewProps {
  mechIds: number[]
  onClose: () => void
  onRemove: (id: number) => void
  onAddToList?: (mech: MechListItem) => void
}

const LOC_ORDER = ['HD', 'CT', 'LT', 'RT', 'LA', 'RA', 'LL', 'RL', 'FLL', 'FRL', 'RLL', 'RRL']
const LOC_NAMES: Record<string, string> = {
  HD: 'Head', CT: 'Center Torso', LT: 'Left Torso', RT: 'Right Torso',
  LA: 'Left Arm', RA: 'Right Arm', LL: 'Left Leg', RL: 'Right Leg',
  FLL: 'Front Left Leg', FRL: 'Front Right Leg', RLL: 'Rear Left Leg', RRL: 'Rear Right Leg',
}

type StatKey = 'tonnage' | 'battle_value' | 'game_damage' | 'tmm' | 'armor_coverage_pct' | 'walk_mp' | 'armor_total' | 'combat_rating' | 'bv_efficiency'

const COMPARE_STATS: { key: StatKey; label: string; format: (v: number | undefined) => string; higherBetter: boolean }[] = [
  { key: 'tonnage', label: 'Tonnage', format: v => v != null ? `${v}t` : '—', higherBetter: false },
  { key: 'battle_value', label: 'BV', format: v => v != null ? String(v) : '—', higherBetter: false },
  { key: 'combat_rating', label: 'Combat Rating', format: v => v != null && v > 0 ? v.toFixed(2) : '—', higherBetter: true },
  { key: 'bv_efficiency', label: 'BV Efficiency', format: v => v != null && v > 0 ? v.toFixed(2) : '—', higherBetter: true },
  { key: 'game_damage', label: 'Game Damage', format: v => v != null && v > 0 ? v.toFixed(1) : '—', higherBetter: true },
  { key: 'walk_mp', label: 'Walk MP', format: v => v != null ? String(v) : '—', higherBetter: true },
  { key: 'tmm', label: 'TMM', format: v => v != null ? `+${v}` : '—', higherBetter: true },
  { key: 'armor_total', label: 'Armor', format: v => v != null ? String(v) : '—', higherBetter: true },
  { key: 'armor_coverage_pct', label: 'Armor %', format: v => v != null ? `${v.toFixed(1)}%` : '—', higherBetter: true },
]

function getStatValue(mech: MechDetail, key: StatKey): number | undefined {
  switch (key) {
    case 'tonnage': return mech.tonnage
    case 'battle_value': return mech.battle_value
    case 'game_damage': return mech.game_damage
    case 'tmm': return mech.stats?.tmm ?? mech.tmm
    case 'armor_coverage_pct': return mech.stats?.armor_coverage_pct ?? mech.armor_coverage_pct
    case 'walk_mp': return mech.stats?.walk_mp ?? mech.walk_mp
    case 'armor_total': return mech.stats?.armor_total ?? mech.armor_total
    case 'combat_rating': return mech.stats?.combat_rating ?? mech.combat_rating
    case 'bv_efficiency': return mech.bv_efficiency
  }
}

/** Color for best/worst stat values */
function statColor(value: number | undefined, best: number | null, worst: number | null, count: number): string | undefined {
  if (value == null || count < 2) return undefined
  if (value === best) return '#4ade80'
  if (value === worst && best !== worst) return '#f87171'
  return undefined
}

/** Simple bar as inline element */
function MiniBar({ value, max, color }: { value: number; max: number; color: string }) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0
  return (
    <div className="w-full h-1.5 rounded-full mt-0.5" style={{ background: 'var(--bg-elevated)' }}>
      <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, background: color }} />
    </div>
  )
}

export function CompareView({ mechIds, onClose, onRemove, onAddToList }: CompareViewProps) {
  const [mechs, setMechs] = useState<(MechDetail | null)[]>(mechIds.map(() => null))
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    Promise.all(mechIds.map(id => fetchMech(id).catch(() => null)))
      .then(results => { setMechs(results); setLoading(false) })
  }, [mechIds])

  const loaded = mechs.filter((m): m is MechDetail => m !== null)
  const colCount = loaded.length

  // Responsive: use CSS class for mobile stacking
  const gridCols = `100px repeat(${colCount}, minmax(120px, 1fr))`
  const weaponGridCols = `repeat(${colCount}, 1fr)`

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto" style={{ background: 'rgba(0,0,0,0.4)' }}>
      <div className="compare-modal rounded-lg shadow-2xl m-4 mt-8 max-w-[1200px] w-full max-h-[90vh] overflow-auto"
        style={{ background: 'var(--bg-page)', border: '1px solid var(--border-default)' }}>

        {/* Header */}
        <div className="sticky top-0 px-5 py-3 flex items-center justify-between z-10"
          style={{ background: 'var(--bg-page)', borderBottom: '1px solid var(--border-default)' }}>
          <h2 className="text-lg font-semibold" style={{ color: 'var(--text-primary)' }}>Compare Mechs</h2>
          <button onClick={onClose} className="text-xl cursor-pointer" style={{ color: 'var(--text-tertiary)' }}>✕</button>
        </div>

        {loading ? (
          <div className="p-8 text-center" style={{ color: 'var(--text-secondary)' }}>Loading...</div>
        ) : (
          <div className="p-5" style={{ minWidth: colCount > 2 ? `${colCount * 200 + 140}px` : undefined }}>

            {/* Mech Names */}
            <div className="compare-grid gap-3" style={{ gridTemplateColumns: gridCols }}>
              <div />
              {loaded.map(m => (
                <div key={m.id} className="text-center">
                  <div className="font-semibold text-sm" style={{ color: 'var(--text-primary)' }}>{m.chassis} {m.model_code}</div>
                  <div className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                    {m.tonnage}t · {m.tech_base}
                  </div>
                  <div className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                    {m.role || '—'} · {m.intro_year || '—'}
                  </div>
                  <div className="flex gap-2 justify-center mt-1">
                    {onAddToList && (
                      <button
                        onClick={() => onAddToList(m)}
                        className="text-xs cursor-pointer px-2 py-0.5 rounded"
                        style={{ color: 'var(--accent)', border: '1px solid var(--accent)' }}
                      >
                        + List
                      </button>
                    )}
                    <button
                      onClick={() => onRemove(m.id)}
                      className="text-xs cursor-pointer px-2 py-0.5 rounded"
                      style={{ color: 'var(--text-tertiary)', border: '1px solid var(--border-default)' }}
                    >
                      Remove
                    </button>
                  </div>
                </div>
              ))}
            </div>

            {/* Stat Comparison */}
            <div className="mt-4 rounded overflow-hidden" style={{ border: '1px solid var(--border-default)' }}>
              {COMPARE_STATS.map((stat, i) => {
                const values = loaded.map(m => getStatValue(m, stat.key))
                const nums = values.filter((v): v is number => v != null && v > 0)
                const best = nums.length > 1 ? (stat.higherBetter ? Math.max(...nums) : Math.min(...nums)) : null
                const worst = nums.length > 1 ? (stat.higherBetter ? Math.min(...nums) : Math.max(...nums)) : null
                const maxVal = nums.length > 0 ? Math.max(...nums) : 0
                const showBar = ['armor_total', 'game_damage', 'combat_rating'].includes(stat.key)

                return (
                  <div
                    key={stat.key}
                    className="compare-grid gap-3 px-3 py-2"
                    style={{
                      gridTemplateColumns: gridCols,
                      background: i % 2 === 0 ? 'var(--bg-surface)' : undefined,
                    }}
                  >
                    <div className="text-xs font-medium self-center" style={{ color: 'var(--text-secondary)' }}>
                      {stat.label}
                    </div>
                    {values.map((v, j) => {
                      const c = statColor(v, best, worst, nums.length)
                      return (
                        <div key={loaded[j].id} className="text-center">
                          <div
                            className="text-sm tabular-nums"
                            style={{ color: c || 'var(--text-primary)', fontWeight: v === best && nums.length > 1 ? 700 : undefined }}
                          >
                            {stat.format(v)}
                          </div>
                          {showBar && v != null && v > 0 && (
                            <MiniBar value={v} max={maxVal} color={c || 'var(--text-tertiary)'} />
                          )}
                        </div>
                      )
                    })}
                  </div>
                )
              })}
            </div>

            {/* Movement */}
            <div className="mt-4">
              <h3 className="text-sm font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>Movement</h3>
              <div className="compare-grid gap-3" style={{ gridTemplateColumns: gridCols }}>
                <div className="text-xs" style={{ color: 'var(--text-secondary)' }}>Walk/Run/Jump</div>
                {loaded.map(m => {
                  const s = m.stats
                  return (
                    <div key={m.id} className="text-center text-sm tabular-nums" style={{ color: 'var(--text-primary)' }}>
                      {s ? `${s.walk_mp}/${s.run_mp}/${s.jump_mp}` : '—'}
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Engine & Heat */}
            <div className="mt-4">
              <h3 className="text-sm font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>Engine & Heat</h3>
              <div className="rounded overflow-hidden" style={{ border: '1px solid var(--border-default)' }}>
                {[
                  { label: 'Engine', fn: (m: MechDetail) => m.stats ? `${m.stats.engine_rating} ${m.stats.engine_type}` : '—' },
                  { label: 'Heat Sinks', fn: (m: MechDetail) => m.stats ? `${m.stats.heat_sink_count} ${m.stats.heat_sink_type}` : '—' },
                ].map((row, i) => (
                  <div
                    key={row.label}
                    className="compare-grid gap-3 px-3 py-2"
                    style={{
                      gridTemplateColumns: gridCols,
                      background: i % 2 === 0 ? 'var(--bg-surface)' : undefined,
                    }}
                  >
                    <div className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>{row.label}</div>
                    {loaded.map(m => (
                      <div key={m.id} className="text-center text-sm" style={{ color: 'var(--text-primary)' }}>{row.fn(m)}</div>
                    ))}
                  </div>
                ))}
              </div>
            </div>

            {/* Weapons with Range Brackets */}
            <div className="mt-4">
              <h3 className="text-sm font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>Weapons & Range</h3>
              <div className="compare-weapon-grid gap-3" style={{ gridTemplateColumns: weaponGridCols }}>
                {loaded.map(m => {
                  const weapons = (m.equipment ?? []).filter(eq => eq.type === 'weapon' || eq.damage)
                  const eqByLoc = (m.equipment ?? []).reduce<Record<string, typeof m.equipment>>((acc, eq) => {
                    const loc = eq.location || '?'
                    if (!acc[loc]) acc[loc] = []
                    acc[loc]!.push(eq)
                    return acc
                  }, {})

                  // Compute range coverage summary
                  const maxShort = Math.max(0, ...weapons.map(w => (w.short_range ?? 0) * (w.quantity ?? 1)))
                  const maxMedium = Math.max(0, ...weapons.map(w => (w.medium_range ?? 0) * (w.quantity ?? 1)))
                  const maxLong = Math.max(0, ...weapons.map(w => (w.long_range ?? 0) * (w.quantity ?? 1)))

                  return (
                    <div key={m.id} className="rounded p-2 text-xs"
                      style={{ border: '1px solid var(--border-default)', color: 'var(--text-primary)' }}>
                      <div className="font-medium text-center mb-2" style={{ color: 'var(--text-secondary)' }}>
                        {m.chassis} {m.model_code}
                      </div>

                      {/* Range coverage summary */}
                      {weapons.length > 0 && (
                        <div className="mb-2 p-1.5 rounded" style={{ background: 'var(--bg-elevated)' }}>
                          <div className="text-center mb-1" style={{ fontSize: '0.65rem', color: 'var(--text-tertiary)', textTransform: 'uppercase' }}>
                            Max Range Reach
                          </div>
                          <div className="flex justify-around text-center">
                            <div><div style={{ color: '#4ade80', fontWeight: 600 }}>{maxShort || '—'}</div><div style={{ color: 'var(--text-tertiary)', fontSize: '0.6rem' }}>Short</div></div>
                            <div><div style={{ color: '#facc15', fontWeight: 600 }}>{maxMedium || '—'}</div><div style={{ color: 'var(--text-tertiary)', fontSize: '0.6rem' }}>Med</div></div>
                            <div><div style={{ color: '#f87171', fontWeight: 600 }}>{maxLong || '—'}</div><div style={{ color: 'var(--text-tertiary)', fontSize: '0.6rem' }}>Long</div></div>
                          </div>
                        </div>
                      )}

                      {/* Weapons by location with range */}
                      {LOC_ORDER.filter(l => eqByLoc[l]).map(loc => (
                        <div key={loc} className="mb-1.5">
                          <div className="uppercase" style={{ fontSize: '0.65rem', color: 'var(--text-tertiary)' }}>
                            {LOC_NAMES[loc] || loc}
                          </div>
                          {eqByLoc[loc]!.map((eq, i) => (
                            <div key={i} className="pl-1">
                              <div className="flex justify-between">
                                <span>{eq.quantity > 1 ? `${eq.quantity}× ` : ''}{eq.name}</span>
                                <span className="tabular-nums ml-2" style={{ color: 'var(--text-tertiary)' }}>
                                  {eq.damage && eq.damage > 0 ? `${eq.damage}` : eq.rack_size ? `${eq.rack_size * 2}` : ''}
                                  {eq.heat ? `/${eq.heat}h` : ''}
                                </span>
                              </div>
                              {(eq.short_range || eq.medium_range || eq.long_range) && (
                                <div className="flex gap-1 pl-1 tabular-nums" style={{ fontSize: '0.6rem', color: 'var(--text-tertiary)' }}>
                                  <span style={{ color: '#4ade80' }}>{eq.short_range ?? '—'}</span>
                                  <span>/</span>
                                  <span style={{ color: '#facc15' }}>{eq.medium_range ?? '—'}</span>
                                  <span>/</span>
                                  <span style={{ color: '#f87171' }}>{eq.long_range ?? '—'}</span>
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      ))}
                      {Object.keys(eqByLoc).length === 0 && (
                        <div className="text-center" style={{ color: 'var(--text-tertiary)' }}>No weapons linked</div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Mobile responsive styles */}
      <style>{`
        .compare-grid {
          display: grid;
        }
        .compare-weapon-grid {
          display: grid;
        }
        @media (max-width: 767px) {
          .compare-modal {
            margin: 0 !important;
            margin-top: 0 !important;
            max-height: 100vh !important;
            border-radius: 0 !important;
            max-width: 100% !important;
          }
          .compare-modal > div:last-child {
            min-width: unset !important;
          }
          .compare-grid {
            grid-template-columns: 1fr !important;
            gap: 0.25rem !important;
          }
          .compare-grid > div:first-child {
            font-weight: 600;
            padding-top: 0.5rem;
          }
          .compare-weapon-grid {
            grid-template-columns: 1fr !important;
          }
        }
      `}</style>
    </div>
  )
}
