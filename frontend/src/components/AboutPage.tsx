import { useEffect, useRef, useState } from 'react'

interface AboutPageProps {
  onClose: () => void
}

export function AboutPage({ onClose }: AboutPageProps) {
  const [visible, setVisible] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)

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

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  return (
    <div className="fixed inset-0 z-50" style={{ background: 'rgba(0,0,0,0.4)' }}>
      <div
        ref={panelRef}
        className="absolute inset-0 sm:inset-auto sm:right-0 sm:top-0 sm:h-full sm:w-[520px] sm:max-w-full shadow-2xl overflow-y-auto transition-transform duration-200 ease-out"
        style={{
          transform: visible ? 'translateX(0)' : 'translateX(100%)',
          background: 'var(--bg-page)',
          borderLeft: '1px solid var(--border-default)',
        }}
      >
        <div className="flex flex-col">
          {/* Header */}
          <div className="p-5 pb-3 flex justify-between items-start" style={{ borderBottom: '1px solid var(--border-default)' }}>
            <div>
              <h2 className="text-xl font-bold" style={{ color: 'var(--text-primary)' }}>About SLIC</h2>
              <p className="text-sm mt-0.5" style={{ color: 'var(--text-secondary)' }}>Star League Intelligence Command</p>
            </div>
            <button
              onClick={onClose}
              className="text-lg cursor-pointer min-w-[44px] min-h-[44px] flex items-center justify-center"
              style={{ color: 'var(--text-tertiary)' }}
            >
              ✕
            </button>
          </div>

          <div className="p-5 space-y-6">
            {/* What is SLIC */}
            <Section title="What is SLIC?">
              <p>
                SLIC is a BattleTech mech database, combat rating system, and list builder.
                Browse 4,200+ mech variants, compare their stats, and build tournament-legal
                force lists with automatic BV calculation.
              </p>
              <p className="mt-2">
                The centerpiece is the <Strong>Combat Rating</Strong> — a 1-10 score derived
                from thousands of Monte Carlo combat simulations that measures how each mech
                actually performs in a fight, not just on paper.
              </p>
            </Section>

            {/* Combat Rating */}
            <Section title="Combat Rating">
              <p>
                Every mech variant is scored by fighting 1,000 simulated duels against a
                baseline opponent: the <Strong>HBK-4P Hunchback</Strong>, which anchors the
                scale at <Strong>5.0</Strong>.
              </p>

              <SubSection title="How It Works">
                <p>
                  Each sim runs on a full 2D hex grid using real MegaMek map boards (192
                  official 16×17 boards, paired side-by-side for a 32×17 play area). Both
                  mechs use tactical AI with minimax decision-making — the first mover
                  evaluates the opponent's best response before committing.
                </p>
              </SubSection>

              <SubSection title="What's Modeled">
                <ul className="list-none space-y-1">
                  <Li>Full BattleTech damage system — 2d6 hit locations, damage transfer through destroyed locations</Li>
                  <Li>Critical hits — slot-based system, engine/gyro/ammo effects, CASE</Li>
                  <Li>Heat management — full heat scale with movement penalties, to-hit modifiers, shutdown, and ammo cook-off</Li>
                  <Li>EV-based weapon selection using the complete heat scale</Li>
                  <Li>Movement & terrain — walking, running, jumping, hex-based LOS</Li>
                  <Li>Physical attacks — punches, kicks, hatchets when they outdamage ranged</Li>
                  <Li>Piloting skill rolls — 20+ damage triggers, falling, leg/gyro damage</Li>
                  <Li>Flanking — probability of rear arc hits based on speed advantage</Li>
                  <Li>Forced withdrawal per BMM p.81</Li>
                  <Li>Equipment — targeting computers, AMS, Artemis IV/V, LBX cluster, UAC jams</Li>
                  <Li>Engine types — IS XL, Clan XL, Light, XXL with correct destruction rules</Li>
                  <Li>Structure types — Reinforced, Composite, Endo Steel</Li>
                </ul>
              </SubSection>

              <SubSection title="The Scale">
                <div className="rounded overflow-hidden" style={{ border: '1px solid var(--border-default)' }}>
                  <div className="grid grid-cols-2 text-xs">
                    <ScaleRow rating="10" label="Godlike" example="" />
                    <ScaleRow rating="9" label="Dominant" example="" />
                    <ScaleRow rating="8" label="Elite" example="Dire Wolf Prime" />
                    <ScaleRow rating="7" label="Excellent" example="Atlas AS7-D, Timber Wolf Prime" />
                    <ScaleRow rating="6" label="Strong" example="Flashman FLS-8K, Mad Cat MKII" />
                    <ScaleRow rating="5" label="Baseline" example="Hunchback HBK-4P" highlight />
                    <ScaleRow rating="4" label="Below Average" example="Phoenix Hawk PXH-1" />
                    <ScaleRow rating="3" label="Weak" example="Commando COM-2D" />
                    <ScaleRow rating="2" label="Poor" example="Locust LCT-1V" />
                    <ScaleRow rating="1" label="Minimal" example="Flea FLE-4" />
                  </div>
                </div>
                <p className="text-xs mt-2" style={{ color: 'var(--text-tertiary)' }}>
                  Formula: 5.0 + 3.5 × ln(defense_turns / offense_turns / baseline_ratio), clamped 1-10.
                  "Offense turns" = median turns to destroy the HBK-4P. "Defense turns" = median turns the HBK-4P takes to destroy you.
                </p>
              </SubSection>

              <SubSection title="Limitations">
                <p>
                  Combat Rating measures raw 1v1 combat effectiveness. It doesn't account for:
                </p>
                <ul className="list-none space-y-1 mt-1">
                  <Li>Lance/Star synergy or combined arms tactics</Li>
                  <Li>Scouting value, indirect fire support, or C3 networks</Li>
                  <Li>Scenario objectives or terrain-specific advantages</Li>
                  <Li>Special munitions or pilot special abilities</Li>
                </ul>
                <p className="mt-2">
                  A mech with a lower CR can still be invaluable in the right role or composition.
                  Use CR as one input alongside BV efficiency, role needs, and your own judgment.
                </p>
              </SubSection>
            </Section>

            {/* BV Efficiency */}
            <Section title="BV Efficiency">
              <p>
                BV Efficiency measures combat value per BV spent. The formula is:
              </p>
              <div className="my-3 px-4 py-2.5 rounded text-sm font-mono text-center" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)' }}>
                BV Efficiency = CR² / (BV / 1000)
              </div>
              <p>
                Squaring the Combat Rating rewards strong mechs disproportionately — a CR 8
                mech isn't just twice as good as a CR 4, it's four times more efficient per BV.
                This helps identify mechs that punch above their weight class in
                tournament list building.
              </p>
            </Section>

            {/* List Builder */}
            <Section title="List Builder">
              <p>
                Build tournament-legal force lists with real-time BV tracking. Features:
              </p>
              <ul className="list-none space-y-1 mt-2">
                <Li>Per-pilot gunnery/piloting skill selection (0-7)</Li>
                <Li>Automatic BV adjustment using the full multiplier table from Total Warfare</Li>
                <Li>Budget tracking with preset buttons (7k, 7.5k, 10k BV)</Li>
                <Li>Validation warnings for unit count (3-6) and chassis limits (max 3 of any chassis)</Li>
                <Li>Save/load lists to browser storage</Li>
                <Li>Export to clipboard for tournament registration</Li>
              </ul>
              <p className="mt-2 text-xs" style={{ color: 'var(--text-tertiary)' }}>
                Keyboard shortcut: press <Strong>L</Strong> to toggle the list builder.
              </p>
            </Section>

            {/* Data Sources */}
            <Section title="Data Sources">
              <ul className="list-none space-y-1">
                <Li><Strong>MegaMek</Strong> — Primary mech data (variants, equipment, stats)</Li>
                <Li><Strong>Master Unit List</Strong> — Era/faction availability, battle values</Li>
                <Li><Strong>Sarna.net</Strong> — Lore links and chassis information</Li>
                <Li><Strong>Iron Wind Metals</Strong> — Miniature availability links</Li>
              </ul>
              <p className="mt-2">
                The database contains {'\u{2248}'}4,200 variants across all eras from Age of War through ilClan,
                covering both Inner Sphere and Clan tech bases.
              </p>
            </Section>

            {/* Source */}
            <Section title="Source Code">
              <p>
                SLIC is open source. Contributions, bug reports, and feature requests are welcome.
              </p>
              <a
                href="https://github.com/JustinWhittecar/slic"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 mt-2 text-sm px-3 py-2 rounded font-medium transition-colors"
                style={{ border: '1px solid var(--border-default)', color: 'var(--text-primary)', background: 'var(--bg-surface)' }}
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/></svg>
                GitHub
                <span style={{ fontSize: 9, opacity: 0.6 }}>↗</span>
              </a>
            </Section>

            {/* Disclaimer */}
            <div className="rounded p-4" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)' }}>
              <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                SLIC is an unofficial fan project and is not affiliated with or endorsed by
                Catalyst Game Labs, The Topps Company, Inc., or any of their subsidiaries.
                BattleTech and MechWarrior are registered trademarks of The Topps Company, Inc.
                All Rights Reserved.
              </p>
              <p className="text-xs mt-2" style={{ color: 'var(--text-tertiary)' }}>
                Built with unreasonable enthusiasm for giant stompy robots.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-sm font-bold uppercase tracking-wide mb-2" style={{ color: 'var(--text-primary)' }}>{title}</h3>
      <div className="text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
        {children}
      </div>
    </div>
  )
}

function SubSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mt-3">
      <h4 className="text-xs font-semibold uppercase tracking-wide mb-1.5" style={{ color: 'var(--text-primary)' }}>{title}</h4>
      <div className="text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
        {children}
      </div>
    </div>
  )
}

function Strong({ children }: { children: React.ReactNode }) {
  return <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{children}</span>
}

function Li({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex gap-2">
      <span style={{ color: 'var(--accent)' }}>•</span>
      <span>{children}</span>
    </li>
  )
}

function ScaleRow({ rating, label, example, highlight }: { rating: string; label: string; example: string; highlight?: boolean }) {
  return (
    <>
      <div
        className="px-3 py-1.5 flex items-center gap-2"
        style={{
          borderBottom: '1px solid var(--border-subtle)',
          background: highlight ? 'var(--bg-elevated)' : undefined,
        }}
      >
        <span className="font-bold tabular-nums" style={{ color: highlight ? 'var(--accent)' : 'var(--text-primary)', minWidth: 20 }}>{rating}</span>
        <span style={{ color: 'var(--text-primary)' }}>{label}</span>
      </div>
      <div
        className="px-3 py-1.5 flex items-center"
        style={{
          borderBottom: '1px solid var(--border-subtle)',
          color: 'var(--text-tertiary)',
          background: highlight ? 'var(--bg-elevated)' : undefined,
        }}
      >
        {example || '—'}
      </div>
    </>
  )
}
