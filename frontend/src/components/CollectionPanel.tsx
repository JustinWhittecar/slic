import { useState, useEffect, useRef, useCallback } from 'react'
import {
  fetchModels, fetchCollection, updateCollection,
  type ChassisModels,
} from '../api/client'

interface CollectionPanelProps {
  onClose: () => void
}

const WEIGHT_CLASSES = [
  { label: 'All', min: 0, max: 200 },
  { label: 'Light', min: 20, max: 35 },
  { label: 'Medium', min: 40, max: 55 },
  { label: 'Heavy', min: 60, max: 75 },
  { label: 'Assault', min: 80, max: 100 },
]

const MFG_COLORS: Record<string, string> = {
  'IWM': '#3b82f6',
  'Iron Wind Metals': '#3b82f6',
  'Catalyst': '#22c55e',
  'Catalyst Game Labs': '#22c55e',
  'Proxy': '#9ca3af',
}

function mfgColor(mfg: string): string {
  return MFG_COLORS[mfg] ?? 'var(--text-tertiary)'
}

export function CollectionPanel({ onClose }: CollectionPanelProps) {
  const [allModels, setAllModels] = useState<ChassisModels[]>([])
  const [collection, setCollection] = useState<Map<number, number>>(new Map())
  const [search, setSearch] = useState('')
  const [weightClass, setWeightClass] = useState('All')
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [loading, setLoading] = useState(true)
  const [visible, setVisible] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    Promise.all([fetchModels(), fetchCollection()]).then(([models, coll]) => {
      setAllModels(models)
      const map = new Map<number, number>()
      for (const item of coll) {
        map.set(item.physical_model_id, item.quantity)
      }
      setCollection(map)
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
    return () => setVisible(false)
  }, [])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  const handleQuantityChange = useCallback(async (modelId: number, delta: number) => {
    const current = collection.get(modelId) ?? 0
    const next = Math.max(0, current + delta)
    const newMap = new Map(collection)
    if (next === 0) {
      newMap.delete(modelId)
    } else {
      newMap.set(modelId, next)
    }
    setCollection(newMap)
    try {
      await updateCollection(modelId, next)
    } catch { /* revert on error? */ }
  }, [collection])

  const wc = WEIGHT_CLASSES.find(w => w.label === weightClass)!
  const filtered = allModels.filter(cm => {
    if (search && !cm.chassis_name.toLowerCase().includes(search.toLowerCase())) return false
    if (weightClass !== 'All' && (cm.tonnage < wc.min || cm.tonnage > wc.max)) return false
    return true
  })

  const totalModels = Array.from(collection.values()).reduce((a, b) => a + b, 0)
  const ownedChassisIds = new Set<number>()
  for (const cm of allModels) {
    for (const m of cm.models) {
      if ((collection.get(m.id) ?? 0) > 0) {
        ownedChassisIds.add(cm.chassis_id)
        break
      }
    }
  }

  const toggleExpand = (id: number) => {
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  return (
    <div className="fixed inset-0 z-50" style={{ background: 'rgba(0,0,0,0.4)' }}>
      <div
        ref={panelRef}
        className="absolute inset-0 sm:inset-auto sm:right-0 sm:top-0 sm:h-full sm:w-[480px] sm:max-w-full shadow-2xl overflow-y-auto transition-transform duration-200 ease-out flex flex-col"
        style={{
          transform: visible ? 'translateX(0)' : 'translateX(100%)',
          background: 'var(--bg-page)',
          borderLeft: '1px solid var(--border-default)',
        }}
      >
        {/* Header */}
        <div className="p-4 pb-3 flex-shrink-0" style={{ borderBottom: '1px solid var(--border-default)' }}>
          <div className="flex justify-between items-center mb-3">
            <h2 className="text-lg font-bold" style={{ color: 'var(--text-primary)' }}>My Collection</h2>
            <button onClick={onClose} className="text-lg cursor-pointer min-w-[44px] min-h-[44px] flex items-center justify-center" style={{ color: 'var(--text-tertiary)' }}>✕</button>
          </div>

          {/* Search */}
          <input
            type="text"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search chassis..."
            className="w-full px-3 py-2 rounded text-sm outline-none mb-3"
            style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)', color: 'var(--text-primary)' }}
          />

          {/* Weight pills */}
          <div className="flex gap-1 mb-3">
            {WEIGHT_CLASSES.map(w => (
              <button
                key={w.label}
                onClick={() => setWeightClass(w.label)}
                className="px-2.5 py-1 text-xs rounded cursor-pointer"
                style={{
                  background: weightClass === w.label ? 'var(--accent)' : 'transparent',
                  color: weightClass === w.label ? '#fff' : 'var(--text-secondary)',
                  border: `1px solid ${weightClass === w.label ? 'var(--accent)' : 'var(--border-default)'}`,
                }}
              >
                {w.label}
              </button>
            ))}
          </div>

          {/* Stats */}
          <div className="text-xs" style={{ color: 'var(--text-secondary)' }}>
            <span className="font-semibold" style={{ color: 'var(--accent)' }}>{totalModels}</span> models owned across{' '}
            <span className="font-semibold" style={{ color: 'var(--accent)' }}>{ownedChassisIds.size}</span> chassis
            <div className="mt-1 h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--bg-elevated)' }}>
              <div className="h-full rounded-full transition-all" style={{ width: `${(ownedChassisIds.size / 732) * 100}%`, background: 'var(--accent)' }} />
            </div>
            <div className="mt-0.5 text-right" style={{ color: 'var(--text-tertiary)' }}>{ownedChassisIds.size}/732 chassis</div>
          </div>
        </div>

        {/* Chassis List */}
        <div className="flex-1 overflow-y-auto">
          {loading ? (
            <div className="p-4 text-center text-sm" style={{ color: 'var(--text-tertiary)' }}>Loading...</div>
          ) : filtered.length === 0 ? (
            <div className="p-4 text-center text-sm" style={{ color: 'var(--text-tertiary)' }}>No chassis found</div>
          ) : (
            filtered.map(cm => {
              const chassisQty = cm.models.reduce((sum, m) => sum + (collection.get(m.id) ?? 0), 0)
              const isExpanded = expanded.has(cm.chassis_id)

              return (
                <div key={cm.chassis_id} style={{ borderBottom: '1px solid var(--border-default)' }}>
                  <button
                    onClick={() => toggleExpand(cm.chassis_id)}
                    className="w-full px-4 py-2.5 flex items-center justify-between cursor-pointer text-left"
                    style={{ background: isExpanded ? 'var(--bg-surface)' : 'transparent' }}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-sm font-medium truncate" style={{ color: 'var(--text-primary)' }}>
                        {cm.chassis_name}
                      </span>
                      <span className="text-xs tabular-nums" style={{ color: 'var(--text-tertiary)' }}>{cm.tonnage}t</span>
                      <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>{cm.tech_base}</span>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      {chassisQty > 0 && (
                        <span className="text-xs font-semibold px-1.5 py-0.5 rounded" style={{ background: 'var(--accent)', color: '#fff' }}>
                          {chassisQty}
                        </span>
                      )}
                      <svg className={`w-3 h-3 transition-transform ${isExpanded ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24" style={{ color: 'var(--text-tertiary)' }}>
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </button>

                  {isExpanded && (
                    <div className="px-4 pb-3 space-y-1.5">
                      {cm.models.map(model => {
                        const qty = collection.get(model.id) ?? 0
                        return (
                          <div key={model.id} className="flex items-center justify-between py-1.5 px-2 rounded" style={{ background: 'var(--bg-elevated)' }}>
                            <div className="min-w-0 flex-1">
                              <div className="text-xs font-medium truncate" style={{ color: 'var(--text-primary)' }}>
                                {model.name}
                              </div>
                              <div className="flex items-center gap-1.5 mt-0.5">
                                <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded-full" style={{ color: '#fff', background: mfgColor(model.manufacturer) }}>
                                  {model.manufacturer}
                                </span>
                                {model.sku && <span className="text-[10px]" style={{ color: 'var(--text-tertiary)' }}>{model.sku}</span>}
                              </div>
                            </div>
                            <div className="flex items-center gap-1 shrink-0 ml-2">
                              <button
                                onClick={(e) => { e.stopPropagation(); handleQuantityChange(model.id, -1) }}
                                className="w-7 h-7 rounded flex items-center justify-center cursor-pointer text-sm font-bold"
                                style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)' }}
                              >−</button>
                              <span className="w-6 text-center text-sm tabular-nums font-semibold" style={{ color: qty > 0 ? 'var(--accent)' : 'var(--text-tertiary)' }}>{qty}</span>
                              <button
                                onClick={(e) => { e.stopPropagation(); handleQuantityChange(model.id, 1) }}
                                className="w-7 h-7 rounded flex items-center justify-center cursor-pointer text-sm font-bold"
                                style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)' }}
                              >+</button>
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}
