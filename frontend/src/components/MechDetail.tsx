import { useEffect, useState, useRef } from 'react'
import { fetchMech, type MechDetail as MechDetailType, type MechEquipment } from '../api/client'

interface MechDetailProps {
  mechId: number
  onClose: () => void
}

const LOCATION_ORDER = ['HD', 'CT', 'LT', 'RT', 'LA', 'RA', 'LL', 'RL']
const LOCATION_NAMES: Record<string, string> = {
  HD: 'Head', CT: 'Center Torso', LT: 'Left Torso', RT: 'Right Torso',
  LA: 'Left Arm', RA: 'Right Arm', LL: 'Left Leg', RL: 'Right Leg',
}

function computeDamageByTurn(equipment: MechEquipment[], heatSinkCount: number, heatSinkType: string): number[] {
  const dissipation = heatSinkCount * ((heatSinkType || '').toLowerCase().includes('double') || (heatSinkType || '').toLowerCase().includes('laser') ? 2 : 1)
  const walkHeat = 1
  const availableDissipation = dissipation - walkHeat

  // Weapons only
  const weapons = equipment.filter(e => e.type === 'energy' || e.type === 'ballistic' || e.type === 'missile' || (e.damage && e.damage > 0))

  const results: number[] = []
  const boardSize = 34
  const myWalk = 4 // assume walk
  const oppWalk = 4

  for (let turn = 1; turn <= 12; turn++) {
    // Both walk toward each other
    const closingSpeed = myWalk + oppWalk
    const range = Math.max(1, boardSize - closingSpeed * turn)

    // Select weapons greedily by effective damage per heat, heat-neutral
    let heatBudget = availableDissipation
    let totalDamage = 0

    // Score each weapon for this range
    const weaponScores = weapons.map(w => {
      const sr = w.short_range ?? 3
      const mr = w.medium_range ?? 6
      const lr = w.long_range ?? 9
      const minR = w.min_range ?? 0

      let rangeMod = 0
      let inRange = true
      if (range > lr) { inRange = false }
      else if (range > mr) { rangeMod = 4 }
      else if (range > sr) { rangeMod = 2 }
      else { rangeMod = 0 }

      // Min range penalty
      let minRangePenalty = 0
      if (minR > 0 && range <= minR) {
        minRangePenalty = minR - range + 1
      }

      const toHit = 7 + rangeMod + minRangePenalty + (w.to_hit_modifier ?? 0)
      const hitProb = toHit >= 12 ? (toHit === 12 ? 1/36 : 0) : Math.max(0, (13 - toHit) * (14 - toHit) / 72)
      // Simplified 2d6 probability
      const hitChance = toHit > 12 ? 0 : toHit <= 2 ? 1 : hitProb

      const dmg = (w.damage ?? 0) * (w.quantity ?? 1)
      const expectedDmg = dmg * hitChance
      const heat = (w.heat ?? 0) * (w.quantity ?? 1)
      const effPerHeat = heat > 0 ? expectedDmg / heat : expectedDmg * 100

      return { weapon: w, expectedDmg, heat, effPerHeat, inRange, hitChance }
    }).filter(s => s.inRange && s.expectedDmg > 0)
      .sort((a, b) => b.effPerHeat - a.effPerHeat)

    for (const ws of weaponScores) {
      if (ws.heat <= heatBudget || ws.heat === 0) {
        totalDamage += ws.expectedDmg
        heatBudget -= ws.heat
      }
    }

    results.push(Math.round(totalDamage * 10) / 10)
  }

  return results
}

function DamageSparkline({ data }: { data: number[] }) {
  const max = Math.max(...data, 1)
  const w = 360
  const h = 80
  const barW = 24
  const gap = (w - barW * data.length) / (data.length + 1)

  return (
    <div>
      <div className="text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Damage by Turn</div>
      <svg viewBox={`0 0 ${w} ${h + 16}`} className="w-full" style={{ maxHeight: 96 }}>
        {data.map((d, i) => {
          const barH = (d / max) * h
          const x = gap + i * (barW + gap)
          const y = h - barH
          return (
            <g key={i}>
              <rect x={x} y={y} width={barW} height={barH} rx={2}
                className="fill-blue-500 dark:fill-blue-400" opacity={0.85} />
              {d > 0 && (
                <text x={x + barW / 2} y={y - 2} textAnchor="middle"
                  className="fill-gray-500 dark:fill-gray-400" fontSize={8} fontFamily="monospace">
                  {d.toFixed(0)}
                </text>
              )}
              <text x={x + barW / 2} y={h + 12} textAnchor="middle"
                className="fill-gray-400 dark:fill-gray-500" fontSize={8} fontFamily="monospace">
                {i + 1}
              </text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}

export function MechDetail({ mechId, onClose }: MechDetailProps) {
  const [mech, setMech] = useState<MechDetailType | null>(null)
  const [loading, setLoading] = useState(true)
  const [visible, setVisible] = useState(false)
  const [techOpen, setTechOpen] = useState(false)
  const [tooltip, setTooltip] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setLoading(true)
    fetchMech(mechId).then(d => { setMech(d); setLoading(false) }).catch(() => setLoading(false))
  }, [mechId])

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
    return () => setVisible(false)
  }, [])

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) onClose()
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [onClose])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  const stats = mech?.stats
  const equipment = mech?.equipment ?? []

  const equipByLoc = equipment.reduce<Record<string, MechEquipment[]>>((acc, eq) => {
    const loc = eq.location || 'Unknown'
    if (!acc[loc]) acc[loc] = []
    acc[loc]!.push(eq)
    return acc
  }, {})

  const sortedLocs = Object.keys(equipByLoc).sort((a, b) => {
    const ai = LOCATION_ORDER.indexOf(a)
    const bi = LOCATION_ORDER.indexOf(b)
    return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi)
  })

  const damageByTurn = stats ? computeDamageByTurn(equipment, stats.heat_sink_count, stats.heat_sink_type) : []

  return (
    <div className="fixed inset-0 bg-black/20 dark:bg-black/50 z-50">
      <div
        ref={panelRef}
        className="absolute right-0 top-0 h-full w-[420px] max-w-full bg-white dark:bg-gray-900 shadow-2xl overflow-y-auto transition-transform duration-200 ease-out"
        style={{ transform: visible ? 'translateX(0)' : 'translateX(100%)' }}
      >
        {loading && (
          <div className="flex items-center justify-center h-full">
            <div className="text-sm text-gray-500 dark:text-gray-400">Loading...</div>
          </div>
        )}

        {mech && !loading && (
          <div className="flex flex-col">
            {/* Header */}
            <div className="p-5 pb-3">
              <div className="flex justify-between items-start">
                <div className="min-w-0 flex-1">
                  <h2 className="text-xl font-bold text-gray-900 dark:text-gray-50 leading-tight">
                    {mech.chassis} {mech.model_code}
                  </h2>
                  <div className="text-sm text-gray-500 dark:text-gray-400 mt-0.5 flex flex-wrap gap-x-1.5">
                    <span>{mech.tonnage}t</span>
                    <span>·</span>
                    <span>{mech.tech_base}</span>
                    {mech.role && <><span>·</span><span>{mech.role}</span></>}
                    {mech.era && <><span>·</span><span>{mech.era}</span></>}
                    {mech.intro_year && <><span>·</span><span>{mech.intro_year}</span></>}
                  </div>
                </div>
                <div className="flex items-start gap-2 ml-3 shrink-0">
                  {mech.battle_value && (
                    <div className="text-right">
                      <div className="text-xs text-gray-400 dark:text-gray-500 uppercase tracking-wide">BV</div>
                      <div className="text-lg font-bold tabular-nums text-gray-900 dark:text-gray-50">{mech.battle_value.toLocaleString()}</div>
                    </div>
                  )}
                  <button onClick={onClose} className="text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 text-lg cursor-pointer mt-0.5">✕</button>
                </div>
              </div>
              {mech.sarna_url && (
                <a href={mech.sarna_url} target="_blank" rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 mt-1">
                  Sarna <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
                </a>
              )}
            </div>

            {/* Core Stats Bar */}
            {stats && (
              <div className="border-t border-b border-gray-100 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-800/50 px-5 py-3">
                <div className="grid grid-cols-4 gap-3 text-center">
                  <div>
                    <div className="text-[10px] uppercase tracking-wider text-gray-400 dark:text-gray-500">Move</div>
                    <div className="text-sm font-semibold tabular-nums text-gray-900 dark:text-gray-100">
                      {stats.walk_mp}/{stats.run_mp}/{stats.jump_mp}
                    </div>
                  </div>
                  <div>
                    <div className="text-[10px] uppercase tracking-wider text-gray-400 dark:text-gray-500">TMM</div>
                    <div className="text-sm font-semibold tabular-nums text-gray-900 dark:text-gray-100">
                      +{stats.tmm ?? 0}
                    </div>
                  </div>
                  <div className="relative"
                    onMouseEnter={() => setTooltip(true)} onMouseLeave={() => setTooltip(false)}>
                    <div className="text-[10px] uppercase tracking-wider text-gray-400 dark:text-gray-500">Game Dmg</div>
                    <div className="text-sm font-semibold tabular-nums text-gray-900 dark:text-gray-100">
                      {(mech.game_damage ?? stats.effective_heat_neutral_damage ?? 0).toFixed(1)}
                    </div>
                    {tooltip && (
                      <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 w-52 p-2 text-xs bg-gray-900 dark:bg-gray-700 text-white rounded shadow-lg z-10">
                        Simulated avg damage/turn over 12 turns. 34-hex board, walk approach, heat-neutral weapon selection.
                      </div>
                    )}
                  </div>
                  <div>
                    <div className="text-[10px] uppercase tracking-wider text-gray-400 dark:text-gray-500">Armor</div>
                    <div className="text-sm font-semibold tabular-nums text-gray-900 dark:text-gray-100">
                      {stats.armor_total}
                      {stats.armor_coverage_pct !== undefined && (
                        <span className="text-[10px] font-normal text-gray-400 dark:text-gray-500 ml-0.5">
                          {stats.armor_coverage_pct.toFixed(0)}%
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Technical Details - Collapsible */}
            {stats && (
              <div className="border-b border-gray-100 dark:border-gray-800">
                <button
                  onClick={() => setTechOpen(!techOpen)}
                  className="w-full px-5 py-2.5 flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer"
                >
                  <span>Technical Details</span>
                  <svg className={`w-3.5 h-3.5 transition-transform ${techOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
                {techOpen && (
                  <div className="px-5 pb-3 grid grid-cols-2 gap-x-4 gap-y-1.5 text-sm">
                    <TechRow label="Engine" value={`${stats.engine_rating} ${stats.engine_type}`} />
                    <TechRow label="Heat Sinks"
                      value={`${stats.heat_sink_count} ${stats.heat_sink_type} (${stats.heat_sink_count * ((stats.heat_sink_type || '').toLowerCase().includes('double') ? 2 : 1)} diss.)`} />
                    {stats.structure_type && <TechRow label="Structure" value={stats.structure_type} />}
                    {stats.armor_type && <TechRow label="Armor" value={stats.armor_type} />}
                    {stats.cockpit_type && <TechRow label="Cockpit" value={stats.cockpit_type} />}
                    {stats.gyro_type && <TechRow label="Gyro" value={stats.gyro_type} />}
                    {stats.myomer_type && stats.myomer_type !== 'Standard' && (
                      <TechRow label="Myomer" value={stats.myomer_type} />
                    )}
                  </div>
                )}
              </div>
            )}

            {/* Equipment by Location */}
            {sortedLocs.length > 0 && (
              <div className="px-5 py-3 border-b border-gray-100 dark:border-gray-800">
                <div className="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-2">Equipment</div>
                <div className="space-y-2.5">
                  {sortedLocs.map(loc => (
                    <div key={loc}>
                      <div className="text-[11px] font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wide mb-0.5">
                        {LOCATION_NAMES[loc] || loc}
                      </div>
                      <table className="w-full text-xs">
                        <tbody>
                          {equipByLoc[loc]!.map(eq => (
                            <tr key={eq.id} className="text-gray-700 dark:text-gray-300">
                              <td className="pr-2 py-0.5">
                                {eq.quantity > 1 ? <span className="text-gray-400">{eq.quantity}× </span> : ''}{eq.name}
                              </td>
                              {eq.damage !== undefined && eq.damage > 0 && (
                                <>
                                  <td className="tabular-nums text-right px-1 text-gray-500 dark:text-gray-400 whitespace-nowrap">{eq.damage}d</td>
                                  <td className="tabular-nums text-right px-1 text-gray-500 dark:text-gray-400 whitespace-nowrap">{eq.heat ?? 0}h</td>
                                  <td className="tabular-nums text-right pl-1 text-gray-500 dark:text-gray-400 whitespace-nowrap">
                                    {eq.short_range ?? '—'}/{eq.medium_range ?? '—'}/{eq.long_range ?? '—'}
                                  </td>
                                </>
                              )}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Damage Sparkline */}
            {damageByTurn.some(d => d > 0) && (
              <div className="px-5 py-3">
                <DamageSparkline data={damageByTurn} />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function TechRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="col-span-2 flex justify-between">
      <span className="text-gray-500 dark:text-gray-400">{label}</span>
      <span className="text-gray-900 dark:text-gray-100 tabular-nums">{value}</span>
    </div>
  )
}
