import { useState, useEffect, useCallback } from 'react'
import type { MechListItem } from '../api/client'
import { fetchRecommendations } from '../api/client'

interface Props {
  budget: number
  excludeIds: number[]
  onAdd: (mech: MechListItem) => void
}

export function RecommendationsPanel({ budget, excludeIds, onAdd }: Props) {
  const [mechs, setMechs] = useState<MechListItem[]>([])
  const [loading, setLoading] = useState(false)
  const [techBase, setTechBase] = useState('All')
  const [weightClass, setWeightClass] = useState('All')

  const load = useCallback(async () => {
    if (budget <= 0) return
    setLoading(true)
    try {
      const results = await fetchRecommendations({
        budget,
        tech_base: techBase,
        weight_class: weightClass,
        exclude: excludeIds,
        limit: 10,
      })
      setMechs(results)
    } catch {
      setMechs([])
    } finally {
      setLoading(false)
    }
  }, [budget, techBase, weightClass, excludeIds])

  useEffect(() => { load() }, [load])

  const selectStyle = {
    background: 'var(--bg-surface)',
    border: '1px solid var(--border-default)',
    color: 'var(--text-primary)',
  }

  return (
    <div className="px-4 py-3" style={{ borderTop: '1px solid var(--border-subtle)', background: 'var(--bg-elevated)' }}>
      <div className="flex items-center gap-2 mb-2 flex-wrap">
        <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>
          Suggestions for {budget.toLocaleString()} BV remaining
        </span>
        <select value={techBase} onChange={e => setTechBase(e.target.value)} className="text-xs px-1.5 py-1 rounded" style={selectStyle}>
          <option value="All">All Tech</option>
          <option value="Inner Sphere">Inner Sphere</option>
          <option value="Clan">Clan</option>
        </select>
        <select value={weightClass} onChange={e => setWeightClass(e.target.value)} className="text-xs px-1.5 py-1 rounded" style={selectStyle}>
          <option value="All">All Weights</option>
          <option value="Light">Light (20-35t)</option>
          <option value="Medium">Medium (40-55t)</option>
          <option value="Heavy">Heavy (60-75t)</option>
          <option value="Assault">Assault (80-100t)</option>
        </select>
      </div>
      {loading ? (
        <p className="text-xs" style={{ color: 'var(--text-tertiary)' }}>Loading…</p>
      ) : mechs.length === 0 ? (
        <p className="text-xs" style={{ color: 'var(--text-tertiary)' }}>No mechs fit the remaining budget.</p>
      ) : (
        <div className="grid gap-1">
          {mechs.map(m => (
            <div key={m.id} className="flex items-center gap-2 py-1 px-2 rounded" style={{ background: 'var(--bg-surface)' }}>
              <button
                onClick={() => onAdd(m)}
                className="text-xs px-1.5 py-0.5 rounded cursor-pointer shrink-0"
                style={{ background: 'var(--accent)', color: '#fff' }}
              >
                +
              </button>
              <span className="text-xs font-medium truncate" style={{ color: 'var(--text-primary)' }}>
                {m.chassis} {m.model_code}
              </span>
              <span className="text-[10px] tabular-nums shrink-0" style={{ color: 'var(--text-tertiary)' }}>
                {m.tonnage}t
              </span>
              <span className="text-[10px] tabular-nums shrink-0" style={{ color: 'var(--text-secondary)' }}>
                BV {m.battle_value?.toLocaleString()}
              </span>
              {m.combat_rating != null && m.combat_rating > 0 && (
                <span className="text-[10px] tabular-nums shrink-0" style={{ color: 'var(--accent)' }}>
                  CR {m.combat_rating.toFixed(1)}
                </span>
              )}
              {m.battle_value != null && m.battle_value > 0 && m.combat_rating != null && m.combat_rating > 0 && (
                <span className="text-[10px] tabular-nums shrink-0" style={{ color: 'var(--text-tertiary)' }}>
                  {(m.combat_rating / (m.battle_value / 100)).toFixed(2)} CR/100BV
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
