import { useEffect, useState } from 'react'
import { fetchMech, type MechDetail } from '../api/client'

interface CompareViewProps {
  mechIds: number[]
  onClose: () => void
  onRemove: (id: number) => void
}

const LOC_ORDER = ['HD', 'CT', 'LT', 'RT', 'LA', 'RA', 'LL', 'RL', 'FLL', 'FRL', 'RLL', 'RRL']
const LOC_NAMES: Record<string, string> = {
  HD: 'Head', CT: 'Center Torso', LT: 'Left Torso', RT: 'Right Torso',
  LA: 'Left Arm', RA: 'Right Arm', LL: 'Left Leg', RL: 'Right Leg',
  FLL: 'Front Left Leg', FRL: 'Front Right Leg', RLL: 'Rear Left Leg', RRL: 'Rear Right Leg',
}

type StatKey = 'tonnage' | 'battle_value' | 'game_damage' | 'tmm' | 'armor_coverage_pct' | 'walk_mp' | 'armor_total'

const COMPARE_STATS: { key: StatKey; label: string; format: (v: number | undefined) => string; higherBetter: boolean }[] = [
  { key: 'tonnage', label: 'Tonnage', format: v => v != null ? `${v}t` : '—', higherBetter: false },
  { key: 'battle_value', label: 'BV', format: v => v != null ? String(v) : '—', higherBetter: false },
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
  }
}

export function CompareView({ mechIds, onClose, onRemove }: CompareViewProps) {
  const [mechs, setMechs] = useState<(MechDetail | null)[]>(mechIds.map(() => null))
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    Promise.all(mechIds.map(id => fetchMech(id).catch(() => null)))
      .then(results => { setMechs(results); setLoading(false) })
  }, [mechIds])

  const loaded = mechs.filter((m): m is MechDetail => m !== null)

  return (
    <div className="fixed inset-0 bg-black/30 dark:bg-black/50 z-50 flex items-start justify-center overflow-y-auto">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-2xl m-4 mt-8 max-w-[1200px] w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-5 py-3 flex items-center justify-between z-10">
          <h2 className="text-lg font-semibold">Compare Mechs</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 text-xl cursor-pointer">✕</button>
        </div>

        {loading ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400">Loading...</div>
        ) : (
          <div className="p-5">
            {/* Mech Names */}
            <div className="grid gap-3" style={{ gridTemplateColumns: `140px repeat(${loaded.length}, 1fr)` }}>
              <div />
              {loaded.map(m => (
                <div key={m.id} className="text-center">
                  <div className="font-semibold text-sm">{m.chassis} {m.model_code}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {m.tonnage}t · {m.tech_base}
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {m.role || '—'} · {m.intro_year || '—'}
                  </div>
                  <button
                    onClick={() => onRemove(m.id)}
                    className="text-xs text-red-400 hover:text-red-600 mt-1 cursor-pointer"
                  >
                    Remove
                  </button>
                </div>
              ))}
            </div>

            {/* Stat Comparison */}
            <div className="mt-4 border border-gray-200 dark:border-gray-700 rounded overflow-hidden">
              {COMPARE_STATS.map((stat, i) => {
                const values = loaded.map(m => getStatValue(m, stat.key))
                const numericValues = values.filter((v): v is number => v != null && v > 0)
                const best = numericValues.length > 0
                  ? (stat.higherBetter ? Math.max(...numericValues) : Math.min(...numericValues))
                  : null

                return (
                  <div
                    key={stat.key}
                    className={`grid gap-3 px-3 py-2 ${i % 2 === 0 ? 'bg-gray-50 dark:bg-gray-750' : ''}`}
                    style={{ gridTemplateColumns: `140px repeat(${loaded.length}, 1fr)` }}
                  >
                    <div className="text-xs font-medium text-gray-500 dark:text-gray-400 self-center">
                      {stat.label}
                    </div>
                    {values.map((v, j) => {
                      const isBest = v != null && v === best && numericValues.length > 1
                      return (
                        <div
                          key={loaded[j].id}
                          className={`text-center text-sm tabular-nums ${isBest ? 'font-bold text-green-600 dark:text-green-400' : ''}`}
                        >
                          {stat.format(v)}
                        </div>
                      )
                    })}
                  </div>
                )
              })}
            </div>

            {/* Movement */}
            <div className="mt-4">
              <h3 className="text-sm font-semibold mb-2">Movement</h3>
              <div className="grid gap-3" style={{ gridTemplateColumns: `140px repeat(${loaded.length}, 1fr)` }}>
                <div className="text-xs text-gray-500 dark:text-gray-400">Walk/Run/Jump</div>
                {loaded.map(m => {
                  const s = m.stats
                  return (
                    <div key={m.id} className="text-center text-sm tabular-nums">
                      {s ? `${s.walk_mp}/${s.run_mp}/${s.jump_mp}` : '—'}
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Engine & Heat */}
            <div className="mt-4">
              <h3 className="text-sm font-semibold mb-2">Engine & Heat</h3>
              <div className="border border-gray-200 dark:border-gray-700 rounded overflow-hidden">
                {[
                  { label: 'Engine', fn: (m: MechDetail) => m.stats ? `${m.stats.engine_rating} ${m.stats.engine_type}` : '—' },
                  { label: 'Heat Sinks', fn: (m: MechDetail) => m.stats ? `${m.stats.heat_sink_count} ${m.stats.heat_sink_type}` : '—' },
                ].map((row, i) => (
                  <div
                    key={row.label}
                    className={`grid gap-3 px-3 py-2 ${i % 2 === 0 ? 'bg-gray-50 dark:bg-gray-750' : ''}`}
                    style={{ gridTemplateColumns: `140px repeat(${loaded.length}, 1fr)` }}
                  >
                    <div className="text-xs font-medium text-gray-500 dark:text-gray-400">{row.label}</div>
                    {loaded.map(m => (
                      <div key={m.id} className="text-center text-sm">{row.fn(m)}</div>
                    ))}
                  </div>
                ))}
              </div>
            </div>

            {/* Equipment Side by Side */}
            <div className="mt-4">
              <h3 className="text-sm font-semibold mb-2">Weapons</h3>
              <div className="grid gap-3" style={{ gridTemplateColumns: `repeat(${loaded.length}, 1fr)` }}>
                {loaded.map(m => {
                  const eqByLoc = (m.equipment ?? []).reduce<Record<string, typeof m.equipment>>((acc, eq) => {
                    const loc = eq.location || '?'
                    if (!acc[loc]) acc[loc] = []
                    acc[loc]!.push(eq)
                    return acc
                  }, {})

                  return (
                    <div key={m.id} className="border border-gray-200 dark:border-gray-700 rounded p-2 text-xs">
                      <div className="font-medium text-center mb-2 text-gray-600 dark:text-gray-300">
                        {m.chassis} {m.model_code}
                      </div>
                      {LOC_ORDER.filter(l => eqByLoc[l]).map(loc => (
                        <div key={loc} className="mb-1.5">
                          <div className="text-gray-400 dark:text-gray-500 uppercase" style={{ fontSize: '0.65rem' }}>
                            {LOC_NAMES[loc] || loc}
                          </div>
                          {eqByLoc[loc]!.map((eq, i) => (
                            <div key={i} className="pl-1 flex justify-between">
                              <span>{eq.quantity > 1 ? `${eq.quantity}× ` : ''}{eq.name}</span>
                              <span className="text-gray-400 tabular-nums ml-2">
                                {eq.damage ? `${eq.damage}` : ''}
                                {eq.heat ? `/${eq.heat}h` : ''}
                              </span>
                            </div>
                          ))}
                        </div>
                      ))}
                      {Object.keys(eqByLoc).length === 0 && (
                        <div className="text-gray-400 text-center">No weapons linked</div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
