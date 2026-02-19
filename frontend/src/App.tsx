import { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { track, trackPageView } from './analytics'
import { MechTable } from './components/MechTable'
import { FilterBar } from './components/FilterBar'
import { MechDetail } from './components/MechDetail'
import { CompareView } from './components/CompareView'
import { ThemeToggle } from './components/ThemeToggle'
import { ListBuilder } from './components/ListBuilder'
import { AboutPage } from './components/AboutPage'
import { FeedbackModal } from './components/FeedbackModal'
import { ChangelogPage } from './components/ChangelogPage'
import { ErrorBoundary } from './components/ErrorBoundary'
import { CollectionPanel } from './components/CollectionPanel'
import { AuthProvider, useAuth } from './contexts/AuthContext'
import type { ListMech } from './components/ListBuilder'
import { fetchMechs, fetchMechsByIds, type MechListItem, type MechFilters } from './api/client'

function UserMenu() {
  const { user, loading, login, logout } = useAuth()
  const [open, setOpen] = useState(false)

  if (loading) return null

  if (!user) {
    return (
      <button
        onClick={login}
        className="text-xs px-3 py-1.5 rounded cursor-pointer flex items-center gap-1.5"
        style={{
          background: 'var(--bg-elevated)',
          color: 'var(--text-secondary)',
          border: '1px solid var(--border-default)',
        }}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
        Sign in
      </button>
    )
  }

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 text-xs px-2 py-1 rounded cursor-pointer"
        style={{
          background: 'var(--bg-elevated)',
          border: '1px solid var(--border-default)',
          color: 'var(--text-secondary)',
        }}
      >
        {user.avatar_url && (
          <img src={user.avatar_url} alt="" className="rounded-full" style={{ width: 20, height: 20 }} />
        )}
        <span>{user.display_name || user.email}</span>
      </button>
      {open && (
        <div
          className="absolute right-0 top-full mt-1 rounded shadow-lg py-1 z-50"
          style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', minWidth: 140 }}
        >
          <button
            onClick={() => { setOpen(false); logout() }}
            className="w-full text-left text-xs px-3 py-1.5 cursor-pointer"
            style={{ color: 'var(--text-secondary)' }}
            onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-hover)')}
            onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
          >
            Sign out
          </button>
        </div>
      )}
    </div>
  )
}

let nextEntryId = 1

function AppInner() {
  const [filters, setFilters] = useState<MechFilters>({ engine_types: ['Fusion', 'XL', 'XXL'] })
  const [selectedMechId, setSelectedMechId] = useState<number | null>(null)
  const [compareIds, setCompareIds] = useState<number[]>([])
  const [showCompare, setShowCompare] = useState(false)
  const [count, setCount] = useState(0)
  const [totalCount, setTotalCount] = useState(0)
  const totalFetched = useRef(false)

  useEffect(() => {
    if (!totalFetched.current) {
      totalFetched.current = true
      fetchMechs({}).then(data => setTotalCount(data.length)).catch(() => {})
    }
  }, [])
  const [listMechs, setListMechs] = useState<ListMech[]>([])
  const [showListBuilder, setShowListBuilder] = useState(true)
  const [showAbout, setShowAbout] = useState(false)
  const [showFeedback, setShowFeedback] = useState(false)
  const [showCollection, setShowCollection] = useState(false)
  const [showChangelog, setShowChangelog] = useState(false)
  const [sharedListBanner, setSharedListBanner] = useState<{ count: number; bv: number } | null>(null)
  const { user } = useAuth()

  // Parse shared list from URL on mount
  // Capture URL params immediately (before child effects like FilterBar can overwrite the URL)
  const initialListParams = useMemo(() => {
    const params = new URLSearchParams(window.location.search)
    const listParam = params.get('list')
    const budgetParam = params.get('budget')
    if (!listParam) return null

    const entries = listParam.split('-').slice(0, 24)
    const parsed: { id: number; g: number; p: number }[] = []
    for (const entry of entries) {
      const [idStr, skills] = entry.split('.')
      const id = parseInt(idStr, 10)
      if (isNaN(id)) continue
      const g = skills?.length >= 1 ? parseInt(skills[0], 10) : 4
      const p = skills?.length >= 2 ? parseInt(skills[1], 10) : 4
      parsed.push({ id, g: isNaN(g) ? 4 : Math.min(7, Math.max(0, g)), p: isNaN(p) ? 4 : Math.min(7, Math.max(0, p)) })
    }
    if (parsed.length === 0) return null
    return { parsed, budgetParam }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!initialListParams) return
    const { parsed, budgetParam } = initialListParams
    const ids = parsed.map(e => e.id)

    if (budgetParam) {
      const b = parseInt(budgetParam, 10)
      if (!isNaN(b)) localStorage.setItem('slic-list-budget', String(b))
    }

    fetchMechsByIds(ids).then(mechs => {
      const mechMap = new Map(mechs.map(m => [m.id, m]))
      const listMechs: ListMech[] = []
      for (const entry of parsed) {
        const mechData = mechMap.get(entry.id)
        if (!mechData) continue
        listMechs.push({
          id: `entry-${nextEntryId++}`,
          mechData,
          pilotGunnery: entry.g,
          pilotPiloting: entry.p,
        })
      }
      if (listMechs.length > 0) {
        setListMechs(listMechs)
        setShowListBuilder(true)
        setSharedListBanner({ count: listMechs.length, bv: 0 })
      }
    }).catch(() => {})
  }, [initialListParams])

  // Track page views
  useEffect(() => { trackPageView('home') }, [])
  useEffect(() => { if (showAbout) trackPageView('about') }, [showAbout])
  useEffect(() => { if (showCollection) trackPageView('collection') }, [showCollection])

  const handleCountChange = useCallback((c: number) => setCount(c), [])
  const clearFilters = useCallback(() => setFilters({ engine_types: ['Fusion', 'XL', 'XXL'] }), [])

  const toggleCompare = useCallback((id: number) => {
    setCompareIds(prev => {
      if (prev.includes(id)) return prev.filter(x => x !== id)
      if (prev.length >= 4) return prev
      return [...prev, id]
    })
  }, [])

  const removeFromCompare = useCallback((id: number) => {
    setCompareIds(prev => {
      const next = prev.filter(x => x !== id)
      if (next.length < 2) setShowCompare(false)
      return next
    })
  }, [])

  const addToList = useCallback((mech: MechListItem) => {
    track('list_add_mech', { mech_id: mech.id, mech_name: `${mech.chassis} ${mech.model_code}` })
    setListMechs(prev => [...prev, {
      id: `entry-${nextEntryId++}`,
      mechData: mech,
      pilotGunnery: 4,
      pilotPiloting: 5,
    }])
    setShowListBuilder(true)
  }, [])

  // Keyboard shortcut: L to toggle list builder
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'l' || e.key === 'L') {
        const tag = (e.target as HTMLElement)?.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
        e.preventDefault()
        setShowListBuilder(s => !s)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  return (
    <div className="min-h-screen transition-colors" style={{ background: 'var(--bg-page)', color: 'var(--text-primary)', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif' }}>
      <div className="max-w-[1600px] mx-auto px-4 py-4">
        <header className="mb-4 flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight">SLIC</h1>
            <p className="text-xs hidden sm:block" style={{ color: 'var(--text-tertiary)' }}>BattleTech Mech Database & List Builder</p>
          </div>
          <div className="flex items-center gap-1 sm:gap-2">
            <button
              onClick={() => setShowListBuilder(s => !s)}
              className="text-xs px-3 py-1.5 rounded cursor-pointer flex items-center gap-1.5"
              style={{
                background: showListBuilder ? 'var(--accent)' : 'var(--bg-elevated)',
                color: showListBuilder ? '#fff' : 'var(--text-secondary)',
                border: `1px solid ${showListBuilder ? 'var(--accent)' : 'var(--border-default)'}`,
              }}
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2"/><rect x="9" y="3" width="6" height="4" rx="1"/></svg>
              List{listMechs.length > 0 ? ` (${listMechs.length})` : ''}
            </button>
            {user && (
              <button
                onClick={() => setShowCollection(true)}
                className="text-xs px-3 py-1.5 rounded cursor-pointer flex items-center gap-1.5"
                style={{
                  background: 'var(--bg-elevated)',
                  color: 'var(--text-secondary)',
                  border: '1px solid var(--border-default)',
                }}
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
                Collection
              </button>
            )}
            <button
              onClick={() => setShowFeedback(true)}
              className="text-xs px-3 py-1.5 rounded cursor-pointer flex items-center gap-1.5"
              style={{
                background: 'var(--bg-elevated)',
                color: 'var(--text-secondary)',
                border: '1px solid var(--border-default)',
              }}
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"/></svg>
              Feedback
            </button>
            <button
              onClick={() => setShowChangelog(true)}
              className="text-xs px-3 py-1.5 rounded cursor-pointer"
              style={{
                background: 'var(--bg-elevated)',
                color: 'var(--text-secondary)',
                border: '1px solid var(--border-default)',
              }}
            >
              Changelog
            </button>
            <button
              onClick={() => setShowAbout(true)}
              className="text-xs px-3 py-1.5 rounded cursor-pointer"
              style={{
                background: 'var(--bg-elevated)',
                color: 'var(--text-secondary)',
                border: '1px solid var(--border-default)',
              }}
            >
              About
            </button>
            <UserMenu />
            <ThemeToggle />
          </div>
        </header>

        <FilterBar filters={filters} onFiltersChange={setFilters} />

        {/* Shared list banner */}
        {sharedListBanner && (
          <div className="mb-2 flex items-center gap-3 rounded px-3 py-2" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--accent)' }}>
            <span className="text-xs" style={{ color: 'var(--accent)' }}>
              📋 Viewing shared list ({sharedListBanner.count} mech{sharedListBanner.count !== 1 ? 's' : ''})
            </span>
            <button
              onClick={() => { setSharedListBanner(null); setListMechs([]) }}
              className="text-xs px-2 py-1 rounded cursor-pointer"
              style={{ color: 'var(--text-secondary)', background: 'var(--bg-surface)', border: '1px solid var(--border-default)' }}
            >
              Clear
            </button>
          </div>
        )}

        {/* List Builder - inline */}
        {showListBuilder && (
          <ListBuilder
            mechs={listMechs}
            onMechsChange={setListMechs}
            onClose={() => setShowListBuilder(false)}
          />
        )}

        {compareIds.length > 0 && (
          <div className="mb-2 flex items-center gap-2 rounded px-3 py-2" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)' }}>
            <span className="text-sm font-medium" style={{ color: 'var(--accent)' }}>
              {compareIds.length} mech{compareIds.length !== 1 ? 's' : ''} selected
            </span>
            {compareIds.length >= 2 && (
              <button
                onClick={() => setShowCompare(true)}
                className="text-sm text-white px-3 py-1 rounded cursor-pointer"
                style={{ background: 'var(--accent)' }}
              >
                Compare
              </button>
            )}
            <button
              onClick={() => { setCompareIds([]); setShowCompare(false) }}
              className="text-sm ml-auto cursor-pointer"
              style={{ color: 'var(--text-secondary)' }}
            >
              Clear
            </button>
          </div>
        )}

        <MechTable
          filters={filters}
          onSelectMech={setSelectedMechId}
          selectedMechId={selectedMechId}
          onCountChange={handleCountChange}
          compareIds={compareIds}
          onToggleCompare={toggleCompare}
          onAddToList={addToList}
          onClearFilters={clearFilters}
        />

        <footer className="mt-2 text-xs flex justify-between items-center" style={{ color: 'var(--text-tertiary)' }}>
          <span>Showing {count} of {totalCount} variants</span>
          <span>Data updated: Feb 18, 2026 · 4,227 variants</span>
          <span className="flex gap-2">
            <button onClick={() => setShowChangelog(true)} className="cursor-pointer hover:underline">Changelog</button>
            <span>·</span>
            <button onClick={() => setShowAbout(true)} className="cursor-pointer hover:underline">About</button>
            <span>·</span>
            <span>slic.dev</span>
          </span>
        </footer>
      </div>

      {selectedMechId !== null && !showCompare && (
        <MechDetail mechId={selectedMechId} onClose={() => setSelectedMechId(null)} onAddToList={addToList} />
      )}

      {showCompare && compareIds.length >= 2 && (
        <CompareView
          mechIds={compareIds}
          onClose={() => setShowCompare(false)}
          onRemove={removeFromCompare}
        />
      )}

      {showAbout && <AboutPage onClose={() => setShowAbout(false)} />}
      {showFeedback && <FeedbackModal onClose={() => setShowFeedback(false)} />}
      {showCollection && <CollectionPanel onClose={() => setShowCollection(false)} />}
      {showChangelog && <ChangelogPage onClose={() => setShowChangelog(false)} />}
    </div>
  )
}

export function App() {
  return (
    <ErrorBoundary>
      <AuthProvider>
        <AppInner />
      </AuthProvider>
    </ErrorBoundary>
  )
}
