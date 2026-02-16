import { useEffect, useState, useCallback } from 'react'
import { fetchMechs, type MechListItem, type MechFilters } from '../api/client'

export function MechTable() {
  const [mechs, setMechs] = useState<MechListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filters, setFilters] = useState<MechFilters>({})

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchMechs(filters)
      setMechs(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [filters])

  useEffect(() => { load() }, [load])

  const updateFilter = (key: keyof MechFilters, value: string) => {
    setFilters(prev => ({ ...prev, [key]: value || undefined }))
  }

  return (
    <div>
      <div style={{ display: 'flex', gap: '1rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
        <input
          type="text"
          placeholder="Search by name..."
          onChange={e => updateFilter('name', e.target.value)}
          style={inputStyle}
        />
        <input
          type="number"
          placeholder="Min tons"
          onChange={e => updateFilter('tonnage_min', e.target.value)}
          style={{ ...inputStyle, width: 100 }}
        />
        <input
          type="number"
          placeholder="Max tons"
          onChange={e => updateFilter('tonnage_max', e.target.value)}
          style={{ ...inputStyle, width: 100 }}
        />
        <select onChange={e => updateFilter('era', e.target.value)} style={inputStyle}>
          <option value="">All Eras</option>
          <option value="Age of War">Age of War</option>
          <option value="Star League">Star League</option>
          <option value="Early Succession Wars">Early Succession Wars</option>
          <option value="Late Succession Wars">Late Succession Wars</option>
          <option value="Clan Invasion">Clan Invasion</option>
          <option value="Civil War">Civil War</option>
          <option value="Jihad">Jihad</option>
          <option value="Dark Age">Dark Age</option>
          <option value="ilClan">ilClan</option>
        </select>
      </div>

      {error && <p style={{ color: 'red' }}>Error: {error}</p>}
      {loading && <p>Loading...</p>}

      {!loading && !error && (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ borderBottom: '2px solid #333', textAlign: 'left' }}>
              <th style={thStyle}>Model</th>
              <th style={thStyle}>Chassis</th>
              <th style={thStyle}>Tons</th>
              <th style={thStyle}>Tech</th>
              <th style={thStyle}>BV</th>
              <th style={thStyle}>Year</th>
              <th style={thStyle}>Era</th>
              <th style={thStyle}>Role</th>
            </tr>
          </thead>
          <tbody>
            {mechs.length === 0 && (
              <tr><td colSpan={8} style={{ padding: '2rem', textAlign: 'center', color: '#999' }}>
                No mechs found. Start the backend and load some data!
              </td></tr>
            )}
            {mechs.map(m => (
              <tr key={m.id} style={{ borderBottom: '1px solid #eee' }}>
                <td style={tdStyle}>{m.model_code}</td>
                <td style={tdStyle}>{m.chassis}</td>
                <td style={tdStyle}>{m.tonnage}</td>
                <td style={tdStyle}>{m.tech_base}</td>
                <td style={tdStyle}>{m.battle_value ?? '—'}</td>
                <td style={tdStyle}>{m.intro_year ?? '—'}</td>
                <td style={tdStyle}>{m.era || '—'}</td>
                <td style={tdStyle}>{m.role || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      <p style={{ color: '#999', fontSize: '0.85rem', marginTop: '0.5rem' }}>
        {mechs.length} variant{mechs.length !== 1 ? 's' : ''} shown
      </p>
    </div>
  )
}

const inputStyle: React.CSSProperties = {
  padding: '0.5rem',
  border: '1px solid #ccc',
  borderRadius: 4,
  fontSize: '0.9rem',
}

const thStyle: React.CSSProperties = { padding: '0.5rem' }
const tdStyle: React.CSSProperties = { padding: '0.5rem' }
