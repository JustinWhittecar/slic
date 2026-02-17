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
}

export function ColumnSelector({ columns, visibility, onVisibilityChange, defaultVisibility }: ColumnSelectorProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className="px-3 py-2 border border-gray-200 dark:border-gray-600 rounded text-sm text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center gap-1"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/>
          <rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/>
        </svg>
        Columns
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded shadow-lg z-50 w-56 py-1">
          {columns.map(col => (
            <label key={col.id} className="flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer">
              <input
                type="checkbox"
                checked={visibility[col.id] !== false}
                onChange={() => onVisibilityChange({ ...visibility, [col.id]: !(visibility[col.id] !== false) })}
              />
              {col.label}
            </label>
          ))}
          <div className="border-t border-gray-200 dark:border-gray-700 mt-1 pt-1 px-3">
            <button
              onClick={() => onVisibilityChange(defaultVisibility)}
              className="text-xs text-blue-600 dark:text-blue-400 hover:underline cursor-pointer"
            >
              Reset to defaults
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
