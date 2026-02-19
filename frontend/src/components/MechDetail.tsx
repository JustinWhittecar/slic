import { useEffect, useState, useRef, useCallback, lazy, Suspense } from 'react'
import { fetchMech, fetchCollectionSummary, type MechDetail as MechDetailType, type MechEquipment } from '../api/client'
import { useAuth } from '../contexts/AuthContext'
import { track } from '../analytics'

const CombatReplay = lazy(() => import('./CombatReplay'))

interface MechDetailProps {
  mechId: number
  onClose: () => void
  onAddToList?: (mech: MechDetailType) => void
}

const LOCATION_ORDER = ['HD', 'CT', 'LT', 'RT', 'LA', 'RA', 'LL', 'RL']
const LOCATION_NAMES: Record<string, string> = {
  HD: 'Head', CT: 'Center Torso', LT: 'Left Torso', RT: 'Right Torso',
  LA: 'Left Arm', RA: 'Right Arm', LL: 'Left Leg', RL: 'Right Leg',
}

function computeDamageByTurn(equipment: MechEquipment[], heatSinkCount: number, heatSinkType: string, walkMP: number, hasTC: boolean): number[] {
  const hsLower = (heatSinkType || '').toLowerCase()
  const dissipation = heatSinkCount * (hsLower.includes('double') || hsLower.includes('laser') ? 2 : 1)
  const weapons = equipment.filter(e =>
    e.type === 'energy' || e.type === 'ballistic' || e.type === 'missile' ||
    (e.expected_damage && e.expected_damage > 0))
  const results: number[] = []
  const boardSize = 34
  const refOppWalk = 4
  const refOptLow = 6
  const refOptHigh = 8
  const gunnery = 4

  const hp = (target: number) => {
    if (target > 12) return 0
    if (target <= 2) return 1
    if (target === 12) return 1/36
    return Math.max(0, (13 - target) * (14 - target) / 72)
  }

  // Build sim weapons — MMLs get dual mode, TC applies -1 to energy/ballistic
  const simWeapons = weapons.flatMap(w => {
    let thm = w.to_hit_modifier ?? 0
    // Targeting Computer: -1 for direct-fire weapons (energy, ballistic)
    if (hasTC && (w.type === 'energy' || w.type === 'ballistic')) {
      thm -= 1
    }
    const isArtillery = w.type === 'artillery'
    const rackSize = (w.rack_size ?? 0) * (w.quantity ?? 1)
    const base = {
      expDmg: (w.expected_damage ?? 0) * (w.quantity ?? 1),
      heat: (w.heat ?? 0) * (w.quantity ?? 1),
      minR: w.min_range ?? 0,
      sr: w.short_range ?? 3,
      mr: w.medium_range ?? 6,
      lr: w.long_range ?? 9,
      thm,
      isMML: false,
      isArtillery,
      rackSize,
      srmDmg: 0, srmSR: 3, srmMR: 6, srmLR: 9,
    }
    if ((w.name || '').toUpperCase().includes('MML') && (w.rack_size ?? 0) > 0) {
      base.isMML = true
      base.srmDmg = (w.rack_size! * 2 * 0.58) * (w.quantity ?? 1)
    }
    return [base]
  })

  const calcDmg = (dist: number, baseTarget: number, heatAvail: number, rTMM: number = 0) => {
    const scored = simWeapons.map(w => {
      let bestED = 0

      if (w.isArtillery) {
        // Artillery direct fire (Tac Ops pp. 150-153):
        // Hit: full damage, all 5-pt groups land (no cluster roll).
        // Miss: scatters 1D6 hexes. 1-hex scatter = adjacent = rackSize-10 damage.
        // Expected miss damage = (1/6) * (rackSize - 10)
        if (dist <= w.lr && dist > w.minR) {
          const pHit = hp(baseTarget - rTMM + w.thm)
          const hitDmg = w.rackSize
          const missDmg = (w.rackSize - 10) / 6
          bestED = hitDmg * pHit + missDmg * (1 - pHit)
        }
      } else if (dist <= w.lr && w.lr > 0) {
        // Normal/LRM mode
        let rm = 0
        if (dist > w.mr) rm = 4
        else if (dist > w.sr) rm = 2
        let mrp = 0
        if (w.minR > 0 && dist <= w.minR) mrp = w.minR - dist + 1
        bestED = w.expDmg * hp(baseTarget + rm + w.thm + mrp)
      }

      // MML SRM mode
      if (w.isMML && dist <= w.srmLR) {
        let rm = 0
        if (dist > w.srmMR) rm = 4
        else if (dist > w.srmSR) rm = 2
        const srmED = w.srmDmg * hp(baseTarget + rm + w.thm)
        if (srmED > bestED) bestED = srmED
      }

      if (bestED <= 0) return null
      const dph = w.heat > 0 ? bestED / w.heat : bestED * 100
      return { effDmg: bestED, heat: w.heat, dph }
    }).filter((s): s is NonNullable<typeof s> => s !== null)
      .sort((a, b) => b.dph - a.dph)

    let hb = heatAvail, dmg = 0
    for (const w of scored) {
      if (w.heat === 0) { dmg += w.effDmg; continue }
      if (hb >= w.heat) { dmg += w.effDmg; hb -= w.heat }
    }
    return dmg
  }

  const heatWalking = Math.max(0, dissipation - 1)
  const heatStanding = dissipation

  let mechPos = 0
  let oppPos = boardSize
  for (let turn = 1; turn <= 12; turn++) {
    const curDist = oppPos - mechPos

    // Ref moves first: tries to reach range 6-8
    let refWalked = false
    if (curDist > refOptHigh) {
      oppPos -= refOppWalk
      if (oppPos < mechPos) oppPos = mechPos
      refWalked = true
    } else if (curDist < refOptLow) {
      oppPos += refOppWalk
      if (oppPos > boardSize) oppPos = boardSize
      refWalked = true
    }

    const refTMM = refWalked ? 1 : 0 // tmmFromMP(4) = +1
    const baseWalked = gunnery + 1 + refTMM
    const baseStood = gunnery + 0 + refTMM

    const advPos = Math.min(mechPos + walkMP, oppPos)
    const advDist = Math.max(1, oppPos - advPos)
    const advDmg = calcDmg(advDist, baseWalked, heatWalking, refTMM)

    const standDist = Math.max(1, oppPos - mechPos)
    const standDmg = calcDmg(standDist, baseStood, heatStanding, refTMM)

    const retPos = Math.max(0, mechPos - walkMP)
    const retDist = Math.max(1, oppPos - retPos)
    const retDmg = calcDmg(retDist, baseWalked, heatWalking, refTMM)

    if (advDmg >= standDmg && advDmg >= retDmg) {
      results.push(Math.round(advDmg * 10) / 10)
      mechPos = advPos
    } else if (standDmg >= retDmg) {
      results.push(Math.round(standDmg * 10) / 10)
    } else {
      results.push(Math.round(retDmg * 10) / 10)
      mechPos = retPos
    }
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
      <div className="flex items-center justify-between mb-1">
        <span className="text-[10px]" style={{ color: 'var(--text-tertiary)' }}>
          Expected damage per turn at optimal engagement range. Gaps = out of range.
        </span>
      </div>
      <svg viewBox={`0 0 ${w} ${h + 28}`} className="w-full" style={{ maxHeight: 110 }}>
        {data.map((d, i) => {
          const topPad = 12
          const barH = d > 0 ? (d / max) * h : 0
          const x = gap + i * (barW + gap)
          const y = topPad + h - barH
          return (
            <g key={i}>
              {d > 0 ? (
                <>
                  <rect x={x} y={y} width={barW} height={barH} rx={2}
                    fill="var(--accent)" opacity={0.75} />
                  <text x={x + barW / 2} y={y - 3} textAnchor="middle"
                    fill="var(--text-secondary)" fontSize={8} fontFamily="monospace">
                    {d.toFixed(0)}
                  </text>
                </>
              ) : (
                <text x={x + barW / 2} y={topPad + h - 4} textAnchor="middle"
                  fill="var(--text-tertiary)" fontSize={8} opacity={0.5}>
                  —
                </text>
              )}
              <text x={x + barW / 2} y={topPad + h + 14} textAnchor="middle"
                fill="var(--text-tertiary)" fontSize={8} fontFamily="monospace">
                {i + 1}
              </text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}

export function MechDetail({ mechId, onClose, onAddToList }: MechDetailProps) {
  const { user } = useAuth()
  const [mech, setMech] = useState<MechDetailType | null>(null)
  const [loading, setLoading] = useState(true)
  const [visible, setVisible] = useState(false)
  const [modelsOpen, setModelsOpen] = useState(true)
  const [techOpen, setTechOpen] = useState(false)
  const [equipOpen, setEquipOpen] = useState(true)
  const [sparkOpen, setSparkOpen] = useState(true)
  const [combatOpen, setCombatOpen] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)
  const [ownedCount, setOwnedCount] = useState(0)

  const [error, setError] = useState(false)
  const [copiedLink, setCopiedLink] = useState(false)

  const loadMech = useCallback(() => {
    setLoading(true)
    setError(false)
    fetchMech(mechId).then(d => { setMech(d); setLoading(false) }).catch(() => { setError(true); setLoading(false) })
  }, [mechId])

  useEffect(() => { loadMech() }, [loadMech])

  useEffect(() => {
    if (mech) {
      track('mech_view', { mech_id: mech.id, mech_name: `${mech.chassis} ${mech.model_code}`, chassis: mech.chassis })
    }
  }, [mech])

  useEffect(() => {
    if (!user || !mech) return
    fetchCollectionSummary().then(summary => {
      const match = summary.find(s => s.chassis_name === mech.chassis)
      setOwnedCount(match?.total_quantity ?? 0)
    }).catch(() => {})
  }, [user, mech])

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

  const damageByTurn = stats ? computeDamageByTurn(equipment, stats.heat_sink_count, stats.heat_sink_type, stats.walk_mp, stats.has_targeting_computer ?? false) : []

  return (
    <div className="fixed inset-0 z-50" style={{ background: 'rgba(0,0,0,0.4)' }}>
      <div
        ref={panelRef}
        className="absolute inset-0 sm:inset-auto sm:right-0 sm:top-0 sm:h-full sm:w-[420px] sm:max-w-full shadow-2xl overflow-y-auto transition-transform duration-200 ease-out"
        style={{
          transform: visible ? 'translateX(0)' : 'translateX(100%)',
          background: 'var(--bg-page)',
          borderLeft: '1px solid var(--border-default)',
        }}
      >
        {loading && (
          <div className="p-5 space-y-4">
            <div className="flex justify-between items-start">
              <div className="space-y-2 flex-1">
                <div className="h-6 w-48 rounded animate-pulse" style={{ background: 'var(--bg-elevated)' }} />
                <div className="h-4 w-32 rounded animate-pulse" style={{ background: 'var(--bg-elevated)' }} />
              </div>
              <button onClick={onClose} className="text-lg cursor-pointer min-w-[44px] min-h-[44px] flex items-center justify-center" style={{ color: 'var(--text-tertiary)' }}>✕</button>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-5 gap-3 py-3" style={{ borderTop: '1px solid var(--border-default)', borderBottom: '1px solid var(--border-default)' }}>
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex flex-col items-center gap-1">
                  <div className="h-3 w-12 rounded animate-pulse" style={{ background: 'var(--bg-elevated)' }} />
                  <div className="h-5 w-16 rounded animate-pulse" style={{ background: 'var(--bg-elevated)' }} />
                </div>
              ))}
            </div>
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-4 rounded animate-pulse" style={{ width: `${70 - i * 10}%`, background: 'var(--bg-elevated)' }} />
            ))}
          </div>
        )}

        {error && !loading && (
          <div className="flex flex-col items-center justify-center h-full gap-3">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ color: 'var(--text-tertiary)' }}>
              <circle cx="12" cy="12" r="10"/><path d="M12 8v4m0 4h.01"/>
            </svg>
            <div className="text-sm font-medium" style={{ color: 'var(--text-secondary)' }}>Something went wrong</div>
            <button onClick={loadMech} className="text-xs px-4 py-2 rounded cursor-pointer font-medium" style={{ background: 'var(--accent)', color: '#fff' }}>Retry</button>
          </div>
        )}

        {mech && !loading && !error && (
          <div className="flex flex-col">
            {/* Header */}
            <div className="p-5 pb-3">
              <div className="flex justify-between items-start">
                <div className="min-w-0 flex-1">
                  <h2 className="text-xl font-bold leading-tight" style={{ color: 'var(--text-primary)' }}>
                    {mech.chassis} {mech.model_code}
                  </h2>
                  {mech.alternate_name && (
                    <div className="text-sm italic" style={{ color: 'var(--text-tertiary)' }}>
                      aka {mech.alternate_name}
                    </div>
                  )}
                  <div className="text-sm mt-0.5 flex flex-wrap gap-x-1.5" style={{ color: 'var(--text-secondary)' }}>
                    <span>{mech.tonnage}t</span>
                    <span>·</span>
                    <span>{mech.tech_base}</span>
                    {mech.role && <><span>·</span><span>{mech.role}</span></>}
                    {mech.era && <><span>·</span><span>{mech.era}</span></>}
                    {mech.intro_year && <><span>·</span><span>{mech.intro_year}</span></>}
                  </div>
                </div>
                <div className="flex items-center gap-3 ml-4 shrink-0">
                  {mech.battle_value && (
                    <div className="text-right">
                      <div className="text-[10px] uppercase tracking-wider" style={{ color: 'var(--text-tertiary)' }}>BV</div>
                      <div className="text-lg font-bold tabular-nums leading-tight" style={{ color: 'var(--text-primary)' }}>{mech.battle_value.toLocaleString()}</div>
                    </div>
                  )}
                  <div className="flex items-center gap-1.5">
                    {onAddToList && mech && (
                      <button
                        onClick={() => onAddToList(mech)}
                        className="text-xs px-2.5 py-1.5 rounded cursor-pointer font-medium"
                        style={{ background: 'var(--accent)', color: '#fff' }}
                        title="Add to list"
                      >+ List</button>
                    )}
                    <button
                      onClick={() => {
                        navigator.clipboard.writeText(`${window.location.origin}/mech/${mech.id}`)
                        setCopiedLink(true)
                        setTimeout(() => setCopiedLink(false), 2000)
                      }}
                      className="text-xs px-2.5 py-1.5 rounded cursor-pointer"
                      style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)' }}
                      title="Copy link"
                    >{copiedLink ? '✓ Copied' : 'Share'}</button>
                    <button onClick={onClose} className="text-lg cursor-pointer min-w-[36px] min-h-[36px] flex items-center justify-center rounded" style={{ color: 'var(--text-tertiary)' }}>✕</button>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-1.5 mt-2 flex-wrap">
                {mech.sarna_url && (
                  <a href={mech.sarna_url} target="_blank" rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-full font-medium transition-colors"
                    style={{ border: '1px solid #2a7b6f', color: '#2a9d8f', background: 'rgba(42, 157, 143, 0.08)' }}>
                    Sarna <span style={{ fontSize: 9, opacity: 0.6 }}>↗</span>
                  </a>
                )}
              </div>
              {user && ownedCount > 0 && (
                <div className="mt-2 text-xs font-medium px-2 py-1 rounded inline-flex items-center gap-1" style={{ background: 'var(--bg-elevated)', color: 'var(--accent)' }}>
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/></svg>
                  You own {ownedCount} model{ownedCount !== 1 ? 's' : ''} of this chassis
                </div>
              )}
            </div>

            {/* Available Models - Collapsible (first section) */}
            {mech.models && (
              <div style={{ borderTop: '1px solid var(--border-default)', borderBottom: '1px solid var(--border-default)' }}>
                <button
                  onClick={() => setModelsOpen(!modelsOpen)}
                  className="w-full px-5 py-2.5 flex items-center justify-between text-xs font-semibold uppercase tracking-wider cursor-pointer"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  <span>Available Models{mech.models.length > 0 ? ` (${mech.models.length})` : ''}</span>
                  <svg className={`w-3.5 h-3.5 transition-transform ${modelsOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
                {modelsOpen && (
                  <div className="px-5 pb-3">
                    {mech.models.length > 0 ? (
                      <div className="space-y-1.5">
                        {mech.models.map(model => {
                          const mfgColors: Record<string, string> = {
                            'IWM': '#3b82f6',
                            'Catalyst': '#22c55e',
                            'Ral Partha': '#a855f7',
                            'Armorcast': '#f59e0b',
                            'WizKids': '#ef4444',
                            'FASA': '#f97316',
                          }
                          const color = mfgColors[model.manufacturer] || 'var(--text-tertiary)'
                          // Determine the best store link per manufacturer
                          let storeUrl: string | null = null
                          let storeLabel = 'eBay'
                          if (model.manufacturer === 'IWM' && model.source_url) {
                            storeUrl = model.source_url
                            storeLabel = 'Iron Wind Metals'
                          } else if (model.manufacturer === 'Catalyst') {
                            // Link to Catalyst store search by pack name (before the " - " mech name)
                            const packName = model.name.includes(' - ') ? model.name.split(' - ')[0] : model.name
                            const catalystSearch = encodeURIComponent(packName)
                            storeUrl = `https://store.catalystgamelabs.com/search?q=${catalystSearch}`
                            storeLabel = 'Catalyst Store'
                          } else {
                            // Ral Partha, FASA, Armorcast, WizKids, Proxy — all out of print → eBay
                            const ebayQuery = encodeURIComponent(`battletech ${model.name} miniature`)
                            storeUrl = model.manufacturer === 'Proxy' ? null : `https://www.ebay.com/sch/i.html?_nkw=${ebayQuery}`
                            storeLabel = 'eBay'
                          }

                          return (
                            <div key={model.id} className="flex items-start gap-2 text-xs py-1">
                              <span
                                className="px-1.5 py-0.5 rounded text-[10px] font-semibold shrink-0 mt-0.5"
                                style={{ background: color + '20', color, border: `1px solid ${color}40` }}
                              >
                                {model.manufacturer}
                              </span>
                              <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-1.5">
                                  <span className="truncate" style={{ color: 'var(--text-primary)' }}>
                                    {model.name}
                                  </span>
                                  {model.sku && (
                                    <span className="shrink-0 tabular-nums" style={{ color: 'var(--text-tertiary)' }}>
                                      {model.sku}
                                    </span>
                                  )}
                                </div>
                                <div className="flex items-center gap-2 mt-0.5">
                                  {storeUrl && (
                                    <a href={storeUrl} target="_blank" rel="noopener noreferrer"
                                      className="inline-flex items-center gap-0.5"
                                      style={{ color: storeLabel === 'eBay' ? 'var(--text-tertiary)' : 'var(--accent)' }}
                                      onClick={e => e.stopPropagation()}>
                                      <span style={{ fontSize: 10 }}>{storeLabel} ↗</span>
                                    </a>
                                  )}
                                </div>
                              </div>
                            </div>
                          )
                        })}
                      </div>
                    ) : (
                      <div className="text-xs" style={{ color: 'var(--text-tertiary)' }}>
                        No official models available — proxy only
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}

            {/* Technical Details - Collapsible */}
            {stats && (
              <div style={{ borderBottom: '1px solid var(--border-default)' }}>
                <button
                  onClick={() => setTechOpen(!techOpen)}
                  className="w-full px-5 py-2.5 flex items-center justify-between text-xs font-semibold uppercase tracking-wider cursor-pointer"
                  style={{ color: 'var(--text-secondary)' }}
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

            {/* Equipment by Location - Collapsible, single aligned table */}
            {sortedLocs.length > 0 && (
              <div style={{ borderBottom: '1px solid var(--border-default)' }}>
                <button
                  onClick={() => setEquipOpen(!equipOpen)}
                  className="w-full px-5 py-2.5 flex items-center justify-between text-xs font-semibold uppercase tracking-wider cursor-pointer"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  <span>Equipment</span>
                  <svg className={`w-3.5 h-3.5 transition-transform ${equipOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
                {equipOpen && (
                  <div className="px-5 pb-3">
                    <table className="w-full text-xs" style={{ borderCollapse: 'collapse', tableLayout: 'fixed' }}>
                      <colgroup>
                        <col style={{ width: '45%' }} />
                        <col style={{ width: '15%' }} />
                        <col style={{ width: '15%' }} />
                        <col style={{ width: '25%' }} />
                      </colgroup>
                      <thead>
                        <tr style={{ color: 'var(--text-tertiary)', fontSize: '10px', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                          <th className="text-left pr-2 py-0.5 font-normal">Equipment</th>
                          <th className="text-right px-1 py-0.5 font-normal">Dmg</th>
                          <th className="text-right px-1 py-0.5 font-normal">Heat</th>
                          <th className="text-right pl-1 py-0.5 font-normal">S/M/L</th>
                        </tr>
                      </thead>
                      <tbody>
                        {sortedLocs.map(loc => {
                          let rowIdx = 0
                          return [
                            <tr key={`loc-${loc}`}>
                              <td colSpan={4} className="pt-2 pb-0.5">
                                <div className="text-[11px] font-semibold uppercase tracking-wide" style={{ color: 'var(--text-primary)' }}>
                                  {LOCATION_NAMES[loc] || loc}
                                </div>
                              </td>
                            </tr>,
                            ...equipByLoc[loc]!.map(eq => {
                              const bg = rowIdx++ % 2 === 1 ? 'var(--bg-elevated)' : 'transparent'
                              return (
                                <tr key={eq.id} style={{ color: 'var(--text-primary)', background: bg }}>
                                  <td className="pr-2 py-1">
                                    {eq.quantity > 1 ? <span style={{ color: 'var(--text-tertiary)' }}>{eq.quantity}× </span> : ''}{eq.name}
                                  </td>
                                  {(eq.damage !== undefined && eq.damage > 0) || (eq.rack_size && eq.rack_size > 0) ? (
                                    <>
                                      <td className="tabular-nums text-right px-1 whitespace-nowrap" style={{ color: 'var(--text-secondary)' }}>{eq.damage && eq.damage > 0 ? eq.damage : eq.rack_size ? `${eq.rack_size * 2}` : '—'}</td>
                                      <td className="tabular-nums text-right px-1 whitespace-nowrap" style={{ color: 'var(--text-secondary)' }}>{eq.heat ?? 0}</td>
                                      <td className="tabular-nums text-right pl-1 whitespace-nowrap" style={{ color: 'var(--text-secondary)' }}>
                                        {eq.short_range ?? '—'}/{eq.medium_range ?? '—'}/{eq.long_range ?? '—'}
                                      </td>
                                    </>
                                  ) : (
                                    <>
                                      <td></td><td></td><td></td>
                                    </>
                                  )}
                                </tr>
                              )
                            })
                          ]
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}

            {/* Damage Sparkline - Collapsible */}
            {damageByTurn.some(d => d > 0) && (
              <div style={{ borderBottom: '1px solid var(--border-default)' }}>
                <button
                  onClick={() => setSparkOpen(!sparkOpen)}
                  className="w-full px-5 py-2.5 flex items-center justify-between text-xs font-semibold uppercase tracking-wider cursor-pointer"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  <span>Damage by Turn</span>
                  <svg className={`w-3.5 h-3.5 transition-transform ${sparkOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
                {sparkOpen && (
                  <div className="px-5 pb-3">
                    <DamageSparkline data={damageByTurn} />
                  </div>
                )}
              </div>
            )}

            {/* Combat Sim Replay - Collapsible, lazy-loaded */}
            <div style={{ borderBottom: '1px solid var(--border-default)' }}>
              <button
                onClick={() => setCombatOpen(!combatOpen)}
                className="w-full px-5 py-2.5 flex items-center justify-between text-xs font-semibold uppercase tracking-wider cursor-pointer"
                style={{ color: 'var(--text-secondary)' }}
              >
                <span>⚡ Combat Sim</span>
                <svg className={`w-3.5 h-3.5 transition-transform ${combatOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>
              {combatOpen && (
                <Suspense fallback={
                  <div className="px-5 py-4">
                    <div className="h-48 rounded animate-pulse flex items-center justify-center" style={{ background: 'var(--bg-elevated)' }}>
                      <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>Loading...</span>
                    </div>
                  </div>
                }>
                  <CombatReplay mechId={mech.id} />
                </Suspense>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function TechRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="col-span-2 flex justify-between">
      <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
      <span className="tabular-nums" style={{ color: 'var(--text-primary)' }}>{value}</span>
    </div>
  )
}
