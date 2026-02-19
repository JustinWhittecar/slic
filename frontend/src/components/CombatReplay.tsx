import { useEffect, useState, useRef, useCallback, useMemo } from 'react'

// ─── Types ──────────────────────────────────────────────────────────────────

interface ReplayHex {
  col: number; row: number; elevation: number; terrain?: string
}

interface ReplayMechSnapshot {
  name: string; col: number; row: number; facing: number; twist: number
  heat: number; armor: number[]; rearArmor: number[]; is: number[]; maxIS: number[]
  prone?: boolean; shutdown?: boolean; destroyed?: boolean
  engineHits?: number; gyroHits?: number; pilotDmg?: number
  walkMP: number; runMP: number; jumpMP: number
  moveMode: string; hexesMoved: number; forcedWithdrawal?: boolean
}

interface ReplayEvent {
  type: string; actor: string; message: string; detail?: string
}

interface ReplayWeaponFire {
  weapon: string; target: number; roll?: number; hit: boolean
  damage: number; location?: string; crit?: string
}

interface ReplayTurn {
  turn: number; attacker: ReplayMechSnapshot; defender: ReplayMechSnapshot
  events: ReplayEvent[]; weapons?: ReplayWeaponFire[]
}

interface ReplayData {
  attackerName: string; defenderName: string
  boardWidth: number; boardHeight: number
  hexes: ReplayHex[]; turns: ReplayTurn[]; result: string
}

// ─── Constants ──────────────────────────────────────────────────────────────

const HEX_SIZE = 22
const SQRT3 = Math.sqrt(3)
const LOC_NAMES = ['HD', 'CT', 'LT', 'RT', 'LA', 'RA', 'LL', 'RL']

// ─── Hex math (flat-top, odd-q offset, 1-indexed) ──────────────────────────

function hexToPixel(col: number, row: number): { x: number; y: number } {
  const c = col - 1
  const r = row - 1
  const x = c * HEX_SIZE * 1.5 + HEX_SIZE
  const y = r * HEX_SIZE * SQRT3 + (c % 2 === 1 ? HEX_SIZE * SQRT3 / 2 : 0) + HEX_SIZE
  return { x, y }
}

function terrainColor(hex: ReplayHex): string {
  const t = hex.terrain || ''
  const elevBrightness = Math.min(hex.elevation * 3, 20)
  if (t.includes('heavy_woods')) return `rgb(${13 + elevBrightness},${40 + elevBrightness},${24 + elevBrightness})`
  if (t.includes('light_woods')) return `rgb(${26 + elevBrightness},${58 + elevBrightness},${42 + elevBrightness})`
  if (t.includes('water')) return `rgb(${26 + elevBrightness},${42 + elevBrightness},${58 + elevBrightness})`
  if (t.includes('rough')) return `rgb(${42 + elevBrightness},${36 + elevBrightness},${24 + elevBrightness})`
  if (t.includes('building')) return `rgb(${60 + elevBrightness},${60 + elevBrightness},${60 + elevBrightness})`
  if (t.includes('pavement') || t.includes('road')) return `rgb(${50 + elevBrightness},${50 + elevBrightness},${50 + elevBrightness})`
  return `rgb(${30 + elevBrightness},${41 + elevBrightness},${59 + elevBrightness})`
}

function hexPoints(cx: number, cy: number, size: number): string {
  const pts: string[] = []
  for (let i = 0; i < 6; i++) {
    const angle = Math.PI / 180 * (60 * i - 30)
    pts.push(`${cx + size * Math.cos(angle)},${cy + size * Math.sin(angle)}`)
  }
  return pts.join(' ')
}

// ─── Component ──────────────────────────────────────────────────────────────

interface CombatReplayProps {
  mechId: number
}

export function CombatReplay({ mechId }: CombatReplayProps) {
  const [replay, setReplay] = useState<ReplayData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [currentTurn, setCurrentTurn] = useState(0)
  const [playing, setPlaying] = useState(false)
  const [speed, setSpeed] = useState(1)
  const [eventsOpen, setEventsOpen] = useState(false)
  const playRef = useRef(false)
  const intervalRef = useRef<ReturnType<typeof setInterval>>()

  // Weapon fire animation state
  const [activeWeaponFires, setActiveWeaponFires] = useState<Array<{
    id: number; weapon: string; hit: boolean; damage: number
    fromX: number; fromY: number; toX: number; toY: number
    type: string; startTime: number
  }>>([])
  const fireIdRef = useRef(0)

  useEffect(() => {
    setLoading(true)
    setError(false)
    fetch(`/api/variants/${mechId}/replay`)
      .then(r => { if (!r.ok) throw new Error(); return r.json() })
      .then((data: ReplayData) => { setReplay(data); setLoading(false); setCurrentTurn(0) })
      .catch(() => { setError(true); setLoading(false) })
  }, [mechId])

  // Auto-play
  useEffect(() => {
    if (!replay) return
    const timer = setTimeout(() => { setPlaying(true); playRef.current = true }, 800)
    return () => clearTimeout(timer)
  }, [replay])

  // Playback
  useEffect(() => {
    playRef.current = playing
    if (playing && replay) {
      intervalRef.current = setInterval(() => {
        if (!playRef.current) return
        setCurrentTurn(prev => {
          if (prev >= replay.turns.length - 1) {
            setPlaying(false)
            playRef.current = false
            return prev
          }
          return prev + 1
        })
      }, 1500 / speed)
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [playing, speed, replay])

  // Trigger weapon fire animations when turn changes
  useEffect(() => {
    if (!replay || !replay.turns[currentTurn]) return
    const turn = replay.turns[currentTurn]
    const fireEvents = turn.events.filter(e => e.type === 'fire')
    if (fireEvents.length === 0) return

    const atk = turn.attacker
    const def = turn.defender
    const fromPx = hexToPixel(atk.col, atk.row)
    const toPx = hexToPixel(def.col, def.row)

    const fires = fireEvents.map((e, i) => {
      const hit = !e.message.includes('MISS')
      const dmgMatch = e.message.match(/(\d+) dmg/)
      const damage = dmgMatch ? parseInt(dmgMatch[1]) : 0
      const weapon = e.message.split(' (')[0]
      const wType = weapon.toLowerCase().includes('laser') || weapon.toLowerCase().includes('ppc') || weapon.toLowerCase().includes('er ')
        ? 'energy'
        : weapon.toLowerCase().includes('srm') || weapon.toLowerCase().includes('lrm') || weapon.toLowerCase().includes('streak') || weapon.toLowerCase().includes('mml') || weapon.toLowerCase().includes('atm')
          ? 'missile'
          : 'ballistic'
      return {
        id: fireIdRef.current++,
        weapon, hit, damage, type: wType,
        fromX: fromPx.x, fromY: fromPx.y,
        toX: toPx.x + (i - fireEvents.length / 2) * 3,
        toY: toPx.y + (i - fireEvents.length / 2) * 3,
        startTime: Date.now() + i * 150,
      }
    })

    setActiveWeaponFires(fires)
    const timer = setTimeout(() => setActiveWeaponFires([]), 2000)
    return () => clearTimeout(timer)
  }, [currentTurn, replay])

  // Crop bounds
  const bounds = useMemo(() => {
    if (!replay) return { minCol: 1, maxCol: 32, minRow: 1, maxRow: 17 }
    const allCols: number[] = []
    const allRows: number[] = []
    for (const t of replay.turns) {
      allCols.push(t.attacker.col, t.defender.col)
      allRows.push(t.attacker.row, t.defender.row)
    }
    const pad = 4
    return {
      minCol: Math.max(1, Math.min(...allCols) - pad),
      maxCol: Math.min(replay.boardWidth, Math.max(...allCols) + pad),
      minRow: Math.max(1, Math.min(...allRows) - pad),
      maxRow: Math.min(replay.boardHeight, Math.max(...allRows) + pad),
    }
  }, [replay])

  const hexMap = useMemo(() => {
    if (!replay) return new Map<string, ReplayHex>()
    const m = new Map<string, ReplayHex>()
    for (const h of replay.hexes) m.set(`${h.col},${h.row}`, h)
    return m
  }, [replay])

  const togglePlay = useCallback(() => setPlaying(p => !p), [])

  if (loading) {
    return (
      <div className="px-5 py-4">
        <div className="h-48 rounded animate-pulse flex items-center justify-center" style={{ background: 'var(--bg-elevated)' }}>
          <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>Loading combat replay...</span>
        </div>
      </div>
    )
  }

  if (error || !replay || replay.turns.length === 0) {
    return (
      <div className="px-5 py-3">
        <div className="text-xs py-3 text-center rounded" style={{ color: 'var(--text-tertiary)', background: 'var(--bg-elevated)' }}>
          No combat replay available
        </div>
      </div>
    )
  }

  const turn = replay.turns[currentTurn]
  const atk = turn.attacker
  const def = turn.defender

  // SVG viewport based on crop bounds
  const topLeft = hexToPixel(bounds.minCol, bounds.minRow)
  const botRight = hexToPixel(bounds.maxCol, bounds.maxRow)
  const svgPad = HEX_SIZE + 4
  const viewBox = `${topLeft.x - svgPad} ${topLeft.y - svgPad} ${botRight.x - topLeft.x + svgPad * 2} ${botRight.y - topLeft.y + svgPad * 2}`

  const totalAtkerArmor = atk.armor.reduce((s, v) => s + v, 0) + atk.rearArmor.reduce((s, v) => s + v, 0)
  const totalDefArmor = def.armor.reduce((s, v) => s + v, 0) + def.rearArmor.reduce((s, v) => s + v, 0)
  // Calculate initial armor from turn 0
  const t0 = replay.turns[0]
  const initAtkArmor = t0.attacker.armor.reduce((s, v) => s + v, 0) + t0.attacker.rearArmor.reduce((s, v) => s + v, 0)
  const initDefArmor = t0.defender.armor.reduce((s, v) => s + v, 0) + t0.defender.rearArmor.reduce((s, v) => s + v, 0)
  const totalAtkIS = atk.is.reduce((s, v) => s + v, 0)
  const totalDefIS = def.is.reduce((s, v) => s + v, 0)
  const maxAtkIS = atk.maxIS.reduce((s, v) => s + v, 0)
  const maxDefIS = def.maxIS.reduce((s, v) => s + v, 0)

  return (
    <div className="px-5 py-3 space-y-2">
      {/* Hex Grid */}
      <div className="rounded overflow-hidden relative" style={{ background: '#0a0f1a', border: '1px solid var(--border-default)' }}>
        {/* Scanline overlay */}
        <div style={{
          position: 'absolute', inset: 0, pointerEvents: 'none', zIndex: 2,
          background: 'repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.04) 2px, rgba(0,0,0,0.04) 4px)',
          mixBlendMode: 'overlay',
        }} />

        <svg viewBox={viewBox} className="w-full" style={{ maxHeight: 280 }}>
          <defs>
            <filter id="glow-blue" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="3" result="blur" />
              <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
            </filter>
            <filter id="glow-red" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="3" result="blur" />
              <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
            </filter>
          </defs>

          {/* Hex grid */}
          {Array.from({ length: bounds.maxCol - bounds.minCol + 1 }, (_, ci) => {
            const col = bounds.minCol + ci
            return Array.from({ length: bounds.maxRow - bounds.minRow + 1 }, (_, ri) => {
              const row = bounds.minRow + ri
              const { x, y } = hexToPixel(col, row)
              const hex = hexMap.get(`${col},${row}`)
              const fill = hex ? terrainColor(hex) : '#1e293b'
              return (
                <polygon
                  key={`${col}-${row}`}
                  points={hexPoints(x, y, HEX_SIZE)}
                  fill={fill}
                  stroke="#334155"
                  strokeWidth={0.5}
                  opacity={0.9}
                />
              )
            })
          })}

          {/* LOS line */}
          {(() => {
            const ap = hexToPixel(atk.col, atk.row)
            const dp = hexToPixel(def.col, def.row)
            return <line x1={ap.x} y1={ap.y} x2={dp.x} y2={dp.y} stroke="rgba(255,255,255,0.08)" strokeWidth={1} strokeDasharray="3,3" />
          })()}

          {/* Weapon fire animations */}
          {activeWeaponFires.map(fire => {
            const elapsed = Date.now() - fire.startTime
            if (elapsed < 0) return null
            const progress = Math.min(elapsed / 600, 1)
            const opacity = progress < 0.7 ? 1 : 1 - (progress - 0.7) / 0.3

            const color = fire.type === 'energy' ? '#00ff88' : fire.type === 'missile' ? '#ffaa00' : '#ff4444'

            return (
              <g key={fire.id}>
                <line
                  x1={fire.fromX} y1={fire.fromY}
                  x2={fire.fromX + (fire.toX - fire.fromX) * progress}
                  y2={fire.fromY + (fire.toY - fire.fromY) * progress}
                  stroke={color} strokeWidth={fire.type === 'energy' ? 2 : 1.5}
                  opacity={opacity}
                  strokeDasharray={fire.type === 'ballistic' ? '4,2' : fire.type === 'missile' ? '2,3' : 'none'}
                />
                {progress > 0.8 && (
                  <text
                    x={fire.toX + 8} y={fire.toY - 5 - (progress - 0.8) * 30}
                    fill={fire.hit ? color : '#888'} fontSize={8} fontFamily="monospace"
                    opacity={opacity}
                  >
                    {fire.hit ? fire.damage : 'MISS'}
                  </text>
                )}
                {fire.hit && progress > 0.85 && (
                  <circle cx={fire.toX} cy={fire.toY} r={HEX_SIZE * 0.5 * (1 - (progress - 0.85) * 6)}
                    fill="none" stroke={color} strokeWidth={2} opacity={opacity * 0.5} />
                )}
              </g>
            )
          })}

          {/* Defender mech */}
          {(() => {
            const { x, y } = hexToPixel(def.col, def.row)
            const effFacing = ((def.facing + def.twist) % 6 + 6) % 6
            const fAngle = Math.PI / 180 * (60 * effFacing - 90)
            const tipX = x + HEX_SIZE * 0.5 * Math.cos(fAngle)
            const tipY = y + HEX_SIZE * 0.5 * Math.sin(fAngle)
            const leftAngle = fAngle + Math.PI * 0.75
            const rightAngle = fAngle - Math.PI * 0.75
            const sz = HEX_SIZE * 0.4
            return (
              <g opacity={def.destroyed ? 0.3 : 1}>
                <polygon
                  points={`${tipX},${tipY} ${x + sz * Math.cos(leftAngle)},${y + sz * Math.sin(leftAngle)} ${x + sz * Math.cos(rightAngle)},${y + sz * Math.sin(rightAngle)}`}
                  fill="#e94560" stroke="#fff" strokeWidth={1.5} filter="url(#glow-red)"
                />
                <text x={x} y={y + HEX_SIZE * 0.85} textAnchor="middle" fill="#e94560" fontSize={6} fontFamily="monospace" fontWeight="bold">
                  DEF
                </text>
              </g>
            )
          })()}

          {/* Attacker mech */}
          {(() => {
            const { x, y } = hexToPixel(atk.col, atk.row)
            const effFacing = ((atk.facing + atk.twist) % 6 + 6) % 6
            const fAngle = Math.PI / 180 * (60 * effFacing - 90)
            const tipX = x + HEX_SIZE * 0.5 * Math.cos(fAngle)
            const tipY = y + HEX_SIZE * 0.5 * Math.sin(fAngle)
            const leftAngle = fAngle + Math.PI * 0.75
            const rightAngle = fAngle - Math.PI * 0.75
            const sz = HEX_SIZE * 0.4
            return (
              <g opacity={atk.destroyed ? 0.3 : 1}>
                <polygon
                  points={`${tipX},${tipY} ${x + sz * Math.cos(leftAngle)},${y + sz * Math.sin(leftAngle)} ${x + sz * Math.cos(rightAngle)},${y + sz * Math.sin(rightAngle)}`}
                  fill="#4aa3df" stroke="#fff" strokeWidth={1.5} filter="url(#glow-blue)"
                />
                <text x={x} y={y + HEX_SIZE * 0.85} textAnchor="middle" fill="#4aa3df" fontSize={6} fontFamily="monospace" fontWeight="bold">
                  ATK
                </text>
              </g>
            )
          })()}
        </svg>
      </div>

      {/* Controls */}
      <div className="flex items-center gap-2 text-xs">
        <button onClick={() => setCurrentTurn(Math.max(0, currentTurn - 1))}
          className="px-2 py-1 rounded cursor-pointer" style={{ background: 'var(--bg-elevated)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' }}>
          ◀
        </button>
        <button onClick={togglePlay}
          className="px-3 py-1 rounded cursor-pointer font-medium" style={{ background: playing ? '#e94560' : 'var(--accent)', color: '#fff' }}>
          {playing ? '⏸' : '▶'}
        </button>
        <button onClick={() => setCurrentTurn(Math.min(replay.turns.length - 1, currentTurn + 1))}
          className="px-2 py-1 rounded cursor-pointer" style={{ background: 'var(--bg-elevated)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' }}>
          ▶
        </button>

        <input
          type="range" min={0} max={replay.turns.length - 1} value={currentTurn}
          onChange={e => { setCurrentTurn(parseInt(e.target.value)); setPlaying(false) }}
          className="flex-1 h-1 accent-purple-500"
          style={{ accentColor: 'var(--accent)' }}
        />

        <span className="tabular-nums whitespace-nowrap" style={{ color: 'var(--text-secondary)' }}>
          T{turn.turn}/{replay.turns.length}
        </span>

        <select value={speed} onChange={e => setSpeed(parseFloat(e.target.value))}
          className="px-1 py-0.5 rounded text-[10px]"
          style={{ background: 'var(--bg-elevated)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' }}>
          <option value={0.5}>0.5×</option>
          <option value={1}>1×</option>
          <option value={2}>2×</option>
        </select>
      </div>

      {/* Status Bars */}
      <div className="grid grid-cols-2 gap-2">
        <MechStatus mech={atk} label="ATK" color="#4aa3df" totalArmor={totalAtkerArmor} initArmor={initAtkArmor} totalIS={totalAtkIS} maxIS={maxAtkIS} />
        <MechStatus mech={def} label="DEF" color="#e94560" totalArmor={totalDefArmor} initArmor={initDefArmor} totalIS={totalDefIS} maxIS={maxDefIS} />
      </div>

      {/* Result badge */}
      {currentTurn === replay.turns.length - 1 && (
        <div className="text-center text-[10px] py-1.5 rounded font-mono font-bold uppercase tracking-wider"
          style={{
            background: replay.result.includes('defender_destroyed') ? 'rgba(74,163,223,0.15)' : 'rgba(233,69,96,0.15)',
            color: replay.result.includes('defender_destroyed') ? '#4aa3df' : '#e94560',
            border: `1px solid ${replay.result.includes('defender_destroyed') ? 'rgba(74,163,223,0.3)' : 'rgba(233,69,96,0.3)'}`,
          }}>
          {replay.result.replace(/_/g, ' ')}
        </div>
      )}

      {/* Events Log */}
      <div>
        <button
          onClick={() => setEventsOpen(!eventsOpen)}
          className="w-full flex items-center justify-between text-[10px] font-semibold uppercase tracking-wider cursor-pointer py-1"
          style={{ color: 'var(--text-tertiary)' }}
        >
          <span>Events ({turn.events.length})</span>
          <svg className={`w-3 h-3 transition-transform ${eventsOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>
        {eventsOpen && (
          <div className="space-y-0.5 max-h-40 overflow-y-auto" style={{ fontSize: 10 }}>
            {turn.events.map((e, i) => {
              const borderColor = {
                fire: '#ff6b35', move: '#4aa3df', psr: '#ffd700', heat: '#ff4444',
                destroyed: '#e94560', physical: '#00cc66', fall: '#ffa500', info: '#555', crit: '#ff00ff',
              }[e.type] || '#555'
              return (
                <div key={i} className="py-0.5 px-1.5 font-mono" style={{ borderLeft: `2px solid ${borderColor}`, color: e.type === 'info' ? 'var(--text-tertiary)' : 'var(--text-secondary)' }}>
                  <span style={{ color: borderColor }}>{e.actor}:</span> {e.message}
                  {e.detail && <span style={{ color: 'var(--text-tertiary)' }}> ({e.detail})</span>}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

// ─── Mech Status Bar ────────────────────────────────────────────────────────

function MechStatus({ mech, label, color, totalArmor, initArmor, totalIS, maxIS }: {
  mech: ReplayMechSnapshot; label: string; color: string
  totalArmor: number; initArmor: number; totalIS: number; maxIS: number
}) {
  const armorPct = initArmor > 0 ? (totalArmor / initArmor) * 100 : 0
  const isPct = maxIS > 0 ? (totalIS / maxIS) * 100 : 0
  const heatPct = Math.min(mech.heat / 30, 1) * 100

  const flags: string[] = []
  if (mech.prone) flags.push('PRONE')
  if (mech.shutdown) flags.push('SHUTDOWN')
  if (mech.destroyed) flags.push('DESTROYED')
  if (mech.forcedWithdrawal) flags.push('FORCED WD')

  return (
    <div className="p-2 rounded text-[10px] space-y-1" style={{ background: 'var(--bg-elevated)', border: `1px solid ${color}30` }}>
      <div className="flex items-center justify-between">
        <span className="font-bold font-mono" style={{ color }}>{label}</span>
        <span className="font-mono truncate ml-1" style={{ color: 'var(--text-secondary)', maxWidth: 120 }}>{mech.name.split(' ').pop()}</span>
      </div>

      {flags.length > 0 && (
        <div className="font-mono font-bold" style={{ color: '#e94560', fontSize: 9 }}>
          {flags.join(' · ')}
        </div>
      )}

      {/* Armor */}
      <div className="flex items-center gap-1">
        <span style={{ color: 'var(--text-tertiary)', width: 18 }}>ARM</span>
        <div className="flex-1 h-2 rounded-full overflow-hidden" style={{ background: '#1e293b' }}>
          <div className="h-full rounded-full transition-all duration-500" style={{ width: `${armorPct}%`, background: `linear-gradient(90deg, ${color}, ${color}88)` }} />
        </div>
      </div>

      {/* IS */}
      <div className="flex items-center gap-1">
        <span style={{ color: 'var(--text-tertiary)', width: 18 }}>IS</span>
        <div className="flex-1 h-2 rounded-full overflow-hidden" style={{ background: '#1e293b' }}>
          <div className="h-full rounded-full transition-all duration-500" style={{ width: `${isPct}%`, background: 'linear-gradient(90deg, #e94560, #e9456088)' }} />
        </div>
      </div>

      {/* Heat */}
      <div className="flex items-center gap-1">
        <span style={{ color: 'var(--text-tertiary)', width: 18 }}>HT</span>
        <div className="flex-1 h-2 rounded-full overflow-hidden" style={{ background: '#1e293b' }}>
          <div className="h-full rounded-full transition-all duration-500" style={{ width: `${heatPct}%`, background: 'linear-gradient(90deg, #f59e0b, #ef4444)' }} />
        </div>
        <span className="tabular-nums" style={{ color: 'var(--text-tertiary)', width: 14, textAlign: 'right' }}>{mech.heat}</span>
      </div>
    </div>
  )
}

export default CombatReplay
