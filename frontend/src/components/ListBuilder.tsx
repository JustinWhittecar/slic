import { useState, useEffect, useCallback, useMemo } from 'react'
import type { MechListItem } from '../api/client'
import { track } from '../analytics'

function escapeXml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

// BV multipliers indexed by [gunnery][piloting], from Total Warfare
const BV_TABLE: number[][] = [
  // P: 0     1     2     3     4     5     6     7     8
  [2.42, 2.31, 2.21, 2.10, 1.93, 1.75, 1.68, 1.59, 1.50], // G0
  [2.21, 2.11, 2.02, 1.92, 1.76, 1.60, 1.54, 1.46, 1.38], // G1
  [2.05, 1.96, 1.88, 1.79, 1.64, 1.50, 1.44, 1.37, 1.30], // G2
  [1.85, 1.78, 1.71, 1.64, 1.50, 1.35, 1.30, 1.24, 1.18], // G3
  [1.71, 1.64, 1.58, 1.51, 1.39, 1.25, 1.12, 1.01, 0.90], // G4
  [1.52, 1.46, 1.40, 1.34, 1.24, 1.12, 1.01, 0.90, 0.80], // G5
  [1.38, 1.33, 1.28, 1.22, 1.12, 1.00, 0.90, 0.81, 0.72], // G6
  [1.26, 1.21, 1.16, 1.12, 1.03, 0.93, 0.84, 0.76, 0.68], // G7
  [1.14, 1.10, 1.06, 1.01, 0.94, 0.86, 0.78, 0.71, 0.64], // G8
]

export function getBVMultiplier(gunnery: number, piloting: number): number {
  const g = Math.max(0, Math.min(8, gunnery))
  const p = Math.max(0, Math.min(8, piloting))
  return BV_TABLE[g][p]
}

const GUNNERY_OPTIONS = [0, 1, 2, 3, 4, 5, 6, 7]
const PILOTING_OPTIONS = [0, 1, 2, 3, 4, 5, 6, 7]
const BUDGET_PRESETS = [3000, 5000, 7000, 10000]

export interface ListMech {
  id: string
  mechData: MechListItem
  pilotGunnery: number
  pilotPiloting: number
}

interface SavedList {
  name: string
  mechs: ListMech[]
  budget: number
}

function getAdjustedBV(entry: ListMech): number {
  const base = entry.mechData.battle_value ?? 0
  return Math.round(base * getBVMultiplier(entry.pilotGunnery, entry.pilotPiloting))
}

interface ListBuilderProps {
  mechs: ListMech[]
  onMechsChange: (mechs: ListMech[]) => void
  onClose: () => void
}

export function ListBuilder({ mechs, onMechsChange, onClose }: ListBuilderProps) {
  const [budget, setBudget] = useState(() => {
    const saved = localStorage.getItem('slic-list-budget')
    return saved ? parseInt(saved, 10) : 7000
  })
  const [saveLoadOpen, setSaveLoadOpen] = useState(false)
  const [saveName, setSaveName] = useState('')
  const [shareMsg, setShareMsg] = useState('')
  const [mulMsg, setMulMsg] = useState('')

  useEffect(() => {
    localStorage.setItem('slic-list-budget', String(budget))
  }, [budget])

  const totalBV = useMemo(() => mechs.reduce((s, m) => s + getAdjustedBV(m), 0), [mechs])
  const totalTonnage = useMemo(() => mechs.reduce((s, m) => s + m.mechData.tonnage, 0), [mechs])
  const avgRating = useMemo(() => {
    if (mechs.length === 0) return 0
    const sum = mechs.reduce((s, m) => s + (m.mechData.combat_rating ?? 0), 0)
    return sum / mechs.length
  }, [mechs])

  const remaining = budget - totalBV
  const pct = budget > 0 ? Math.min(100, (totalBV / budget) * 100) : 0

  const overBudget = totalBV > budget && budget > 0

  const removeMech = useCallback((id: string) => {
    onMechsChange(mechs.filter(m => m.id !== id))
  }, [mechs, onMechsChange])

  const updatePilot = useCallback((id: string, gunnery: number, piloting: number) => {
    onMechsChange(mechs.map(m => m.id === id ? { ...m, pilotGunnery: gunnery, pilotPiloting: piloting } : m))
  }, [mechs, onMechsChange])

  // Save/Load
  const getSavedLists = (): SavedList[] => {
    try { return JSON.parse(localStorage.getItem('slic-saved-lists') || '[]') } catch { return [] }
  }

  const saveList = () => {
    if (!saveName.trim()) return
    const lists = getSavedLists().filter(l => l.name !== saveName.trim())
    lists.push({ name: saveName.trim(), mechs, budget })
    localStorage.setItem('slic-saved-lists', JSON.stringify(lists))
    setSaveName('')
  }

  const loadList = (list: SavedList) => {
    onMechsChange(list.mechs)
    setBudget(list.budget)
    setSaveLoadOpen(false)
  }

  const deleteList = (name: string) => {
    const lists = getSavedLists().filter(l => l.name !== name)
    localStorage.setItem('slic-saved-lists', JSON.stringify(lists))
    setSaveLoadOpen(s => s)
  }

  const exportMul = () => {
    track('list_export_mul', { mech_count: mechs.length, total_bv: totalBV })
    const entities = mechs.map((entry, i) => {
      const chassis = entry.mechData.chassis
      const model = entry.mechData.model_code
      const g = entry.pilotGunnery
      const p = entry.pilotPiloting
      const type = entry.mechData.config || 'Biped'
      return `    <entity chassis="${escapeXml(chassis)}" model="${escapeXml(model)}" type="${escapeXml(type)}">\n        <pilot name="MechWarrior ${i + 1}" gunnery="${g}" piloting="${p}"/>\n    </entity>`
    })
    const xml = `<?xml version="1.0" encoding="UTF-8"?>\n<unit version="1.0">\n${entities.join('\n')}\n</unit>\n`
    const blob = new Blob([xml], { type: 'application/xml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'slic-force.mul'
    a.click()
    URL.revokeObjectURL(url)
    setMulMsg('Downloaded!')
    setTimeout(() => setMulMsg(''), 2000)
  }

  const shareList = () => {
    track('list_share', { mech_count: mechs.length, total_bv: totalBV })
    const encoded = mechs.map(m =>
      `${m.mechData.id}.${m.pilotGunnery}${m.pilotPiloting}`
    ).join('-')
    const url = new URL(window.location.href)
    url.search = ''
    url.searchParams.set('list', encoded)
    url.searchParams.set('budget', String(budget))
    navigator.clipboard.writeText(url.toString())
    setShareMsg('Link copied!')
    setTimeout(() => setShareMsg(''), 2000)
  }

  return (
    <div className="mb-4 rounded" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)' }}>
      {/* Header row: title, budget, summary stats, actions */}
      <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 px-4 py-2.5" style={{ borderBottom: mechs.length > 0 ? '1px solid var(--border-subtle)' : 'none' }}>
        {/* Row 1 on mobile: title + budget */}
        <div className="flex items-center gap-2 flex-wrap">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold">List Builder</span>
            <button onClick={onClose} className="text-xs cursor-pointer min-w-[44px] min-h-[44px] sm:min-w-0 sm:min-h-0 flex items-center justify-center" style={{ color: 'var(--text-tertiary)' }} title="Close">✕</button>
          </div>

          {/* BV Budget */}
          <div className="flex items-center gap-1.5">
            <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>BV</span>
            <input
              type="number"
              value={budget}
              onChange={e => setBudget(parseInt(e.target.value) || 0)}
              className="w-16 px-1.5 py-0.5 rounded text-xs tabular-nums text-right"
              style={{
                background: 'var(--bg-elevated)',
                border: '1px solid var(--border-default)',
                color: 'var(--text-primary)',
              }}
            />
            <div className="flex gap-0.5">
              {BUDGET_PRESETS.map(p => (
                <button
                  key={p}
                  onClick={() => setBudget(p)}
                  className="text-[10px] px-1.5 py-0.5 rounded cursor-pointer tabular-nums"
                  style={{
                    background: budget === p ? 'var(--accent)' : 'transparent',
                    color: budget === p ? '#fff' : 'var(--text-tertiary)',
                    border: `1px solid ${budget === p ? 'var(--accent)' : 'var(--border-default)'}`,
                  }}
                >
                  {(p/1000).toFixed(1)}k
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Row 2 on mobile: progress + stats */}
        {mechs.length > 0 && (
          <div className="flex items-center gap-2 flex-1 min-w-0 flex-wrap">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <div className="flex-1 h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--bg-elevated)', maxWidth: 200 }}>
                <div
                  className="h-full rounded-full transition-all"
                  style={{
                    width: `${pct}%`,
                    background: remaining < 0 ? '#ef4444' : 'var(--accent)',
                  }}
                />
              </div>
              <span className="text-xs tabular-nums whitespace-nowrap" style={{ color: remaining < 0 ? '#ef4444' : 'var(--text-tertiary)' }}>
                {totalBV} / {budget}
                {remaining >= 0 ? ` (${remaining} left)` : ` (${Math.abs(remaining)} over)`}
              </span>
            </div>
            <div className="flex gap-3 text-xs tabular-nums" style={{ color: 'var(--text-secondary)' }}>
              <span>{mechs.length} mechs</span>
              <span>{totalTonnage}t</span>
              <span>Avg CR {avgRating.toFixed(1)}</span>
            </div>
          </div>
        )}

        {/* Row 3 on mobile: actions */}
        <div className="flex gap-1.5 sm:ml-auto">
          <button
            onClick={exportMul}
            className="text-xs px-2 py-1 rounded cursor-pointer"
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)' }}
            disabled={mechs.length === 0}
            title="Download .mul file"
          >
            {mulMsg || 'Export'}
          </button>
          <button
            onClick={shareList}
            className="text-xs px-2 py-1 rounded cursor-pointer"
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)' }}
            disabled={mechs.length === 0}
          >
            {shareMsg || 'Share'}
          </button>
          <button
            onClick={() => setSaveLoadOpen(!saveLoadOpen)}
            className="text-xs px-2 py-1 rounded cursor-pointer"
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)' }}
          >
            Save/Load
          </button>
          {mechs.length > 0 && (
            <button
              onClick={() => onMechsChange([])}
              className="text-xs px-2 py-1 rounded cursor-pointer"
              style={{ color: 'var(--text-tertiary)' }}
            >
              Clear
            </button>
          )}
        </div>
      </div>

      {/* Over budget warning */}
      {overBudget && (
        <div className="px-4 py-1.5" style={{ borderBottom: '1px solid var(--border-subtle)' }}>
          <span className="text-xs" style={{ color: '#f59e0b' }}>
            Over budget by {(totalBV - budget).toLocaleString()} BV
          </span>
        </div>
      )}

      {/* Mech cards - horizontal scrolling row */}
      {mechs.length > 0 ? (
        <div className="flex gap-2 px-4 py-3 overflow-x-auto">
          {mechs.map((entry) => (
            <MechCard
              key={entry.id}
              entry={entry}
              onRemove={removeMech}
              onUpdatePilot={updatePilot}
            />
          ))}
        </div>
      ) : (
        <div className="px-4 py-4 text-center">
          <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
            Click <span style={{ color: 'var(--accent)' }}>+</span> on any mech below to start building your list
          </p>
        </div>
      )}

      {/* Save/Load panel */}
      {saveLoadOpen && (
        <div className="px-4 py-3" style={{ borderTop: '1px solid var(--border-subtle)', background: 'var(--bg-elevated)' }}>
          <div className="flex flex-col sm:flex-row gap-2 sm:items-center mb-2">
            <div className="flex gap-2 items-center">
              <input
                type="text"
                placeholder="List name…"
                value={saveName}
                onChange={e => setSaveName(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && saveList()}
                className="text-xs px-2 py-1.5 rounded flex-1 sm:w-48 sm:flex-none"
                style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)', color: 'var(--text-primary)' }}
              />
              <button
                onClick={saveList}
                className="text-xs px-2 py-1.5 rounded cursor-pointer"
                style={{ background: 'var(--accent)', color: '#fff' }}
              >
                Save
              </button>
            </div>
            <div className="flex gap-2 flex-wrap">
              {getSavedLists().map(list => (
                <div key={list.name} className="flex items-center gap-1">
                  <button
                    onClick={() => loadList(list)}
                    className="text-xs cursor-pointer px-2 py-1.5 rounded"
                    style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)', color: 'var(--text-primary)' }}
                  >
                    {list.name} <span style={{ color: 'var(--text-tertiary)' }}>({list.mechs.length}, {list.budget} BV)</span>
                  </button>
                  <button
                    onClick={() => deleteList(list.name)}
                    className="text-xs cursor-pointer min-w-[44px] min-h-[44px] sm:min-w-0 sm:min-h-0 flex items-center justify-center"
                    style={{ color: 'var(--text-tertiary)' }}
                  >
                    ×
                  </button>
                </div>
              ))}
              {getSavedLists().length === 0 && (
                <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>No saved lists</span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function MechCard({
  entry, onRemove, onUpdatePilot,
}: {
  entry: ListMech
  onRemove: (id: string) => void
  onUpdatePilot: (id: string, g: number, p: number) => void
}) {
  const adjustedBV = getAdjustedBV(entry)
  const multiplier = getBVMultiplier(entry.pilotGunnery, entry.pilotPiloting)

  return (
    <div
      className="rounded px-3 py-2 flex-shrink-0 group relative"
      style={{
        background: 'var(--bg-page)',
        border: '1px solid var(--border-subtle)',
        minWidth: 180,
        maxWidth: 220,
      }}
    >
      {/* Remove button */}
      <button
        onClick={() => onRemove(entry.id)}
        className="absolute top-1 right-1.5 text-xs cursor-pointer opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity min-w-[44px] min-h-[44px] sm:min-w-0 sm:min-h-0 flex items-center justify-center"
        style={{ color: 'var(--text-tertiary)' }}
      >
        ×
      </button>

      {/* Mech name */}
      <div className="text-xs font-medium truncate pr-4" title={`${entry.mechData.chassis} ${entry.mechData.model_code}`}>
        {entry.mechData.model_code}
      </div>
      <div className="text-[10px] truncate" style={{ color: 'var(--text-tertiary)' }}>
        {entry.mechData.chassis}
      </div>

      {/* Stats row */}
      <div className="flex items-center gap-2 mt-1.5 text-[10px] tabular-nums" style={{ color: 'var(--text-secondary)' }}>
        <span>{entry.mechData.tonnage}t</span>
        <span>BV {adjustedBV}</span>
        {multiplier !== 1.0 && (
          <span style={{ color: 'var(--accent)' }}>×{multiplier.toFixed(2)}</span>
        )}
      </div>

      {/* Pilot skills */}
      <div className="flex items-center gap-1 mt-1.5">
        <div className="flex items-center gap-0.5">
          <span className="text-[9px]" style={{ color: 'var(--text-tertiary)' }}>G</span>
          <select
            value={entry.pilotGunnery}
            onChange={e => onUpdatePilot(entry.id, Number(e.target.value), entry.pilotPiloting)}
            className="text-xs px-1 py-0.5 rounded cursor-pointer tabular-nums"
            style={{
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border-default)',
              color: 'var(--text-secondary)',
              width: '2.25rem',
            }}
          >
            {GUNNERY_OPTIONS.map(g => <option key={g} value={g}>{g}</option>)}
          </select>
        </div>
        <span className="text-[9px]" style={{ color: 'var(--text-tertiary)' }}>/</span>
        <div className="flex items-center gap-0.5">
          <span className="text-[9px]" style={{ color: 'var(--text-tertiary)' }}>P</span>
          <select
            value={entry.pilotPiloting}
            onChange={e => onUpdatePilot(entry.id, entry.pilotGunnery, Number(e.target.value))}
            className="text-xs px-1 py-0.5 rounded cursor-pointer tabular-nums"
            style={{
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border-default)',
              color: 'var(--text-secondary)',
              width: '2.25rem',
            }}
          >
            {PILOTING_OPTIONS.map(p => <option key={p} value={p}>{p}</option>)}
          </select>
        </div>
        {entry.mechData.combat_rating != null && entry.mechData.combat_rating > 0 && (
          <span className="text-[10px] tabular-nums ml-auto" style={{ color: 'var(--text-tertiary)' }}>
            CR {entry.mechData.combat_rating.toFixed(1)}
          </span>
        )}
      </div>
    </div>
  )
}
