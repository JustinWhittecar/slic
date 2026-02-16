# SLIC — BattleTech Mech Database & List Builder

A web app for browsing BattleTech mechs, filtering by era/faction/tonnage, and building lance lists.

## Architecture

- **Backend:** Go 1.22+ with stdlib router and pgx for Postgres
- **Frontend:** React 18 + TypeScript + Vite
- **Database:** PostgreSQL 16

## Quick Start

### 1. Start Postgres

```bash
docker compose up -d
```

### 2. Run the backend

```bash
cd backend
go run ./cmd/server
```

Server starts on http://localhost:8080

### 3. Run the frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend starts on http://localhost:3000 (proxies API to :8080)

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check |
| GET | `/api/mechs` | List mechs (filterable) |
| GET | `/api/mechs/:id` | Mech detail with equipment |

### Query Parameters for `/api/mechs`

- `name` — search by mech or chassis name
- `tonnage_min`, `tonnage_max` — filter by weight class
- `era` — filter by era name
- `faction` — filter by faction name or abbreviation
- `role` — filter by role

## Project Structure

```
slic/
├── backend/          # Go API server
│   ├── cmd/server/   # Entry point
│   └── internal/     # DB, handlers, models, ingestion
├── frontend/         # React + Vite app
├── data/             # Data source documentation
└── docker-compose.yml
```

## Data Sources

See [data/README.md](data/README.md) for details on MegaMek, MUL, Sarna, and miniature sources.
