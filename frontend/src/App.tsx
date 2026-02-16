import { MechTable } from './components/MechTable'

export function App() {
  return (
    <div style={{ maxWidth: 1200, margin: '0 auto', padding: '1rem', fontFamily: 'system-ui, sans-serif' }}>
      <header style={{ marginBottom: '2rem' }}>
        <h1 style={{ margin: 0 }}>SLIC</h1>
        <p style={{ color: '#666', margin: '0.25rem 0 0' }}>BattleTech Mech Database &amp; List Builder</p>
      </header>
      <MechTable />
    </div>
  )
}
