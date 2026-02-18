# SLIC MVP Scope

## What Is SLIC?

A BattleTech mech database, combat rating system, and list builder. The tool serious BT players use to research mechs, understand combat effectiveness, and build tournament-legal lists.

**Core Value Prop**: Combat Rating — a Monte Carlo-simulated 1-10 score for every mech in BattleTech, run across real MegaMek mapsheets with full rules fidelity. No other tool does this.

---

## MVP Definition

**Ship when a BT player can**: Browse all mechs → understand combat effectiveness → compare options → build a tournament list → export it.

---

## MVP Features

### ✅ Done

| Feature | Status | Notes |
|---------|--------|-------|
| Mech database (4,076 variants) | ✅ | MegaMek MTF + MUL data |
| Searchable/filterable table | ✅ | Name, tech base, era, tonnage, engine type |
| Sortable columns | ✅ | All stats sortable |
| Column customization | ✅ | Show/hide columns |
| Key stats | ✅ | BV, TMM, Armor %, HN Damage, Alpha Damage, Optimal Range |
| Combat Rating (v1) | ✅ | Monte Carlo sim, 1-10 scale |
| BV Efficiency | ✅ | CR² / (BV/1000) |
| Detail panel | ✅ | Slide-out with equipment, sparkline, sourcing links |
| Compare mode | ✅ | 2-4 mechs side-by-side |
| List Builder | ✅ | Inline, BV budget, pilot skills, lance/star grouping |
| Pilot skill selectors (G/P) | ✅ | Full 9×9 BV multiplier table |
| Era filtering | ✅ | Cumulative by intro year |
| Engine type filtering | ✅ | Multi-select pills |
| Dark mode | ✅ | Linear/Roam-inspired |
| Sarna/IWM/Catalyst links | ✅ | Sourcing buttons in detail panel |
| Clan/IS dual naming | ✅ | Search and display |
| URL param sync | ✅ | Shareable filter states |

### 🔄 In Progress

| Feature | Status | Notes |
|---------|--------|-------|
| Combat Rating v2 (2D hex grid) | 🔄 | Running overnight, ~5hr ETA |
| Full BT heat scale | 🔄 | Merged in v2 |
| PSRs/falling | 🔄 | Merged in v2 |
| Equipment effects (TC, AMS, Artemis) | 🔄 | Merged in v2 |
| Initiative system | 🔄 | In v2 |
| Real MegaMek mapsheets | 🔄 | 192 official boards |

### 🔲 Needed for MVP

| Feature | Priority | Effort | Notes |
|---------|----------|--------|-------|
| **Deploy to Fly.io** | P0 | Medium | Postgres→SQLite migration, Go serves React via embed.FS |
| **Mobile-responsive layout** | P0 | Small | Table horizontal scroll, filter collapse, detail panel fullscreen |
| **List export (text/copy)** | P0 | Small | Copy list to clipboard for tournament registration |
| **Loading states** | P0 | Tiny | Skeleton/spinner while data loads |
| **Error handling** | P0 | Small | Graceful fallbacks, retry on API failure |
| **About/FAQ page** | P1 | Small | Explain CR methodology, data sources, limitations |
| **List save/load (localStorage)** | P1 | Small | Already partially done? Verify works |
| **Combat Rating tooltip** | P1 | Tiny | Explain what the number means on hover |
| **Favicon + meta tags** | P1 | Tiny | OG tags for sharing links |
| **Performance: virtualized table** | P1 | Medium | 4K rows — may need react-virtualized for smooth scroll |

### 🔮 Post-MVP (Nice to Have)

| Feature | Notes |
|---------|-------|
| **List sharing via URL** | Encode list state in URL params |
| **Saved lists (accounts)** | Requires auth — defer |
| **Force composition rules** | BTCC rules validation (3-6 units, max 3 chassis, etc.) |
| **Mech art/images** | MUL has some, licensing unclear |
| **Compare from list** | Select mechs in list builder → compare |
| **Role tags** | Striker, Brawler, Sniper, Scout — from MUL data |
| **Design quirks** | From BMM, affect combat in nuanced ways |
| **Community ratings** | Let players rate mechs, compare to CR |
| **Tournament list browser** | Import/share competitive lists |
| **Print-friendly list view** | For tournament registration sheets |

---

## Architecture for Deploy

```
┌─────────────────┐
│   Fly.io VM      │
│                   │
│  Go binary        │
│  ├── embed.FS     │ ← React static build
│  ├── SQLite DB    │ ← All mech data + CR scores
│  └── HTTP server  │ ← API + static files
└─────────────────┘
```

- **Single artifact**: Go binary with embedded frontend + SQLite
- **~$2/month** on Fly.io (shared-cpu-1x, 256MB)
- **No external DB** — SQLite embedded, data baked in at build time
- **Read-only at runtime** — CR scores precomputed, no user writes (until accounts)

### Migration Tasks
1. Write SQLite schema (mirror Postgres with minor syntax changes)
2. Export Postgres data → SQLite seed file
3. Update Go DB layer (`pgx` → `modernc.org/sqlite` or `mattn/go-sqlite3`)
4. Add `embed.FS` for React build
5. Dockerfile for Fly.io
6. `fly.toml` config
7. Domain setup (slic.gg? slicbt.com?)

---

## Timeline Estimate

| Phase | Duration | What |
|-------|----------|------|
| CR v2 completion | ✅ Tonight | Full sim run, DB update |
| Deploy pipeline | 1-2 evenings | SQLite migration, embed, Fly.io |
| Mobile responsive | 1 evening | CSS breakpoints, layout tweaks |
| Polish (loading, errors, about) | 1 evening | Small items |
| **MVP Launch** | **~1 week** | Shareable URL, real domain |

---

## What MVP Is NOT

- Not a full game simulator (it's a rating tool)
- Not an account-based platform (no login, no saved state beyond localStorage)
- Not a replacement for MegaMek (it's a research/planning tool)
- Not a tier list (CR measures combat effectiveness, not "best mech to pick")
- Not authoritative for all playstyles (open-field 1v1 baseline, terrain-averaged)

---

## Success Criteria

MVP is successful if:
1. A BT tournament player can find and compare mechs faster than any existing tool
2. Combat Ratings are credible enough that experienced players mostly agree with rankings
3. List Builder produces valid BTCC-format lists
4. The site loads fast and works on phone
5. At least one BT community (Reddit, Discord, forum) finds it useful

---

## Domain Ideas
- `slic.gg` ← short, memorable, gaming TLD
- `slicbt.com` ← clear what it is
- `slic.tools` ← descriptive
- `battlemechlab.com` ← more discoverable but less unique
