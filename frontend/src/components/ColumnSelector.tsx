import { useState, useRef, useEffect } from 'react'
import type { VisibilityState } from '@tanstack/react-table'

interface ColumnDef {
  id: string
  label: string
}

interface ColumnSelectorProps {
  columns: ColumnDef[]
  visibility: VisibilityState
  onVisibilityChange: (v: VisibilityState) => void
  defaultVisibility: VisibilityState
  columnOrder?: string[]
  onColumnOrderChange?: (order: string[]) => void
}

export function ColumnSelector({ columns, visibility, onVisibilityChange, defaultVisibility, columnOrder, onColumnOrderChange }: ColumnSelectorProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const [dragIdx, setDragIdx] = useState<number | null>(null)
  const [dragOverIdx, setDragOverIdx] = useState<number | null>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  // Sort columns by columnOrder if provided
  const orderedColumns = columnOrder
    ? [...columns].sort((a, b) => {
        const ai = columnOrder.indexOf(a.id)
        const bi = columnOrder.indexOf(b.id)
        return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
      })
    : columns

  const handleDrop = (toIdx: number) => {
    if (dragIdx === null || dragIdx === toIdx || !onColumnOrderChange || !columnOrder) return
    const fromCol = orderedColumns[dragIdx]
    const toCol = orderedColumns[toIdx]
    if (fromCol.id === 'name' || toCol.id === 'name') return

    const order = [...columnOrder]
    const fromOrderIdx = order.indexOf(fromCol.id)
    const toOrderIdx = order.indexOf(toCol.id)
    if (fromOrderIdx === -1 || toOrderIdx === -1) return
    order.splice(fromOrderIdx, 1)
    order.splice(toOrderIdx, 0, fromCol.id)
    onColumnOrderChange(order)
    setDragIdx(null)
    setDragOverIdx(null)
  }

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className="px-3 py-2 rounded text-sm cursor-pointer flex items-center gap-1"
        style={{ border: '1px solid var(--border-default)', color: 'var(--text-secondary)' }}
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
          <rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/>
        </svg>
        Columns
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 rounded shadow-lg z-50 w-56 py-1 max-h-80 overflow-y-auto"
          style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)' }}>
          {orderedColumns.map((col, idx) => {
            const isDraggable = col.id !== 'name' && !!onColumnOrderChange
            return (
              <label
                key={col.id}
                draggable={isDraggable}
                onDragStart={() => isDraggable && setDragIdx(idx)}
                onDragOver={e => { e.preventDefault(); setDragOverIdx(idx) }}
                onDragEnd={() => { setDragIdx(null); setDragOverIdx(null) }}
                onDrop={() => handleDrop(idx)}
                className="flex items-center gap-2 px-3 py-1.5 text-sm cursor-pointer"
                style={{
                  color: 'var(--text-primary)',
                  borderTop: dragOverIdx === idx ? '2px solid var(--accent)' : '2px solid transparent',
                  opacity: dragIdx === idx ? 0.4 : 1,
                }}
                onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-hover)')}
                onMouseLeave={e => (e.currentTarget.style.background = '')}
              >
                {isDraggable && <span style={{ color: 'var(--text-tertiary)', fontSize: '10px', cursor: 'grab' }}>⋮⋮</span>}
                <input
                  type="checkbox"
                  checked={visibility[col.id] !== false}
                  onChange={() => onVisibilityChange({ ...visibility, [col.id]: !(visibility[col.id] !== false) })}
                  style={{ accentColor: 'var(--accent)' }}
                />
                {col.label}
              </label>
            )
          })}
          <div className="mt-1 pt-1 px-3" style={{ borderTop: '1px solid var(--border-default)' }}>
            <button
              onClick={() => onVisibilityChange(defaultVisibility)}
              className="text-xs cursor-pointer hover:underline"
              style={{ color: 'var(--accent)' }}
            >
              Reset to defaults
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
