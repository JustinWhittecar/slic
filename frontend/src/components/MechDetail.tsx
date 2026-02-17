import { useEffect, useState, useRef } from 'react'
import { fetchMech, type MechDetail as MechDetailType } from '../api/client'

interface MechDetailProps {
  mechId: number
  onClose: () => void
}

export function MechDetail({ mechId, onClose }: MechDetailProps) {
  const [mech, setMech] = useState<MechDetailType | null>(null)
  const [loading, setLoading] = useState(true)
  const [visible, setVisible] = useState(false)
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

  const stats = mech?.stats
  type EquipItem = NonNullable<NonNullable<typeof mech>['equipment']>[number]
  const equipByLoc = (mech?.equipment ?? []).reduce<Record<string, EquipItem[]>>((acc, eq) => {
    const loc = eq.location || 'Unknown'
    if (!acc[loc]) acc[loc] = []
    acc[loc]!.push(eq)
    return acc
  }, {})

  return (
    <div className="fixed inset-0 bg-black/20 dark:bg-black/40 z-50">
      <div
        ref={panelRef}
        className="absolute right-0 top-0 h-full w-[420px] max-w-full bg-white dark:bg-gray-800 shadow-xl overflow-y-auto transition-transform duration-200 ease-out"
        style={{ transform: visible ? 'translateX(0)' : 'translateX(100%)' }}
      >
        <div className="p-5">
          <div className="flex justify-between items-start mb-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                {mech ? `${mech.chassis} ${mech.model_code}` : '...'}
              </h2>
              {mech && (
                <div className="text-sm text-gray-500 dark:text-gray-400">{mech.tonnage}t · {mech.tech_base}</div>
              )}
            </div>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 text-xl cursor-pointer">✕</button>
          </div>

          {loading && <div className="text-sm text-gray-500 dark:text-gray-400">Loading...</div>}

          {stats && (
            <div className="grid grid-cols-2 gap-3 mb-5">
              <Stat label="Walk/Run/Jump" value={`${stats.walk_mp}/${stats.run_mp}/${stats.jump_mp}`} />
              <Stat label="Armor" value={String(stats.armor_total)} />
              <Stat label="Heat Sinks" value={`${stats.heat_sink_count} ${stats.heat_sink_type}`} />
              <Stat label="Engine" value={`${stats.engine_rating} ${stats.engine_type}`} />
              <Stat label="BV" value={String(mech?.battle_value ?? '—')} />
              {stats.tmm !== undefined && <Stat label="TMM" value={`+${stats.tmm}`} />}
              {stats.armor_coverage_pct !== undefined && <Stat label="Armor Coverage" value={`${stats.armor_coverage_pct.toFixed(1)}%`} />}
              {stats.heat_neutral_damage !== undefined && <Stat label="Heat Neutral Dmg" value={String(stats.heat_neutral_damage.toFixed(1))} />}
            </div>
          )}

          {Object.keys(equipByLoc).length > 0 && (
            <div className="mb-5">
              <h3 className="text-sm font-semibold mb-2">Equipment</h3>
              {Object.entries(equipByLoc).map(([loc, items]) => (
                <div key={loc} className="mb-2">
                  <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">{loc}</div>
                  {items!.map(eq => (
                    <div key={eq.id} className="text-sm pl-2">
                      {eq.quantity > 1 ? `${eq.quantity}× ` : ''}{eq.name}
                      {eq.damage ? ` (${eq.damage} dmg)` : ''}
                    </div>
                  ))}
                </div>
              ))}
            </div>
          )}

          <div className="border-t border-gray-200 dark:border-gray-700 pt-3">
            <div className="text-xs text-gray-400 dark:text-gray-500">Lore section coming soon.</div>
          </div>
        </div>
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-gray-50 dark:bg-gray-700 rounded p-2">
      <div className="text-xs text-gray-500 dark:text-gray-400">{label}</div>
      <div className="text-sm font-medium tabular-nums">{value}</div>
    </div>
  )
}
