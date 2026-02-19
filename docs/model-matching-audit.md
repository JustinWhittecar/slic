# Physical Model-Chassis Matching Audit

**Date**: 2026-02-19
**Branch**: `fix/model-matching`

## Summary

The physical model to chassis matching is in **excellent shape**. All 1,622 models are matched to chassis, with no orphaned records.

## Statistics

| Metric | Count |
|--------|-------|
| Total physical models | 1,622 |
| Matched to chassis | 1,622 (100%) |
| Unmatched | 0 |
| Total chassis | 732 |
| Total variants | 4,227 |

### Models by Manufacturer

| Manufacturer | Count |
|-------------|-------|
| Proxy | 732 (1 per chassis) |
| IWM | 408 |
| Catalyst | 205 |
| Ral Partha | 274 |
| Armorcast | 3 |

## Spelling Variation Matches (Correct)

These 9 models have names that differ from their chassis but are correctly matched:

- Wolf Trap "Tora" WFT-1 → Wolf Trap (Tora)
- Man O War "Gargoyle" Prime → Man O' War
- Wardog (20-797) → War Dog
- Wolftrap (20-811) → Wolf Trap (Tora)
- Blackwatch (20-903) → Black Watch
- O'Bakemono (20-904) → O-Bakemono
- Rajin (20-906) → Raijin
- Menshen (20-941) → Men Shen
- No-Daichi (20-693) → No-Dachi

## Coverage Gaps (Proxy-Only Chassis)

These popular chassis (by variant count) have no real miniatures — only proxy entries. These are **market gaps**, not matching failures:

| Chassis | Tonnage | Variants |
|---------|---------|----------|
| Crosscut | 30 | 18 |
| Carbine | 30 | 13 |
| Grand Dragon | 60 | 12 |
| Brigand | 25 | 11 |
| Star Adder | 90 | 10 |
| Grigori | 60 | 10 |
| Celerity | 15 | 10 |
| Deva | 70 | 9 |

## Fixes Applied

### 1. SQLite Schema: Added missing `material` and `year` columns

The `physical_models` table in `slic.db` was missing `material TEXT` and `year INTEGER` columns that exist in the Postgres source and are referenced by the export script and `mechs_sqlite.go` handler. Added via `ALTER TABLE`.

### 2. Models Handler: Added `material` and `year` to API response

Updated `backend/internal/handlers/models.go`:
- Added `Material` and `Year` fields to `PhysicalModel` struct
- Updated SQL query to select these columns
- Updated `rows.Scan` to populate them

This brings the `/api/models` endpoint in line with the `/api/mechs/:id` endpoint which already served these fields.

## Conclusion

The matching logic (which runs in Postgres during ingestion) is working correctly. No mismatches found. The main issues were schema drift between Postgres and SQLite, and incomplete API field exposure.
