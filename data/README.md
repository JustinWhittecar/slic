# Data Sources

## MegaMek
- `.mtf` files contain individual mech variant definitions
- Source: https://github.com/MegaMek/megamek (under `megamek/data/mechfiles/`)
- Parser: `backend/internal/ingestion/megamek.go`

## Master Unit List (MUL)
- Official CGL database: https://masterunitlist.info/
- Has era/faction availability data, BV, intro years
- No public API — may need scraping or manual data entry

## Sarna.net
- BattleTech wiki with detailed lore and specs: https://www.sarna.net/wiki/
- Good for chassis descriptions, history, images
- Link stored in `chassis.sarna_url`

## Iron Wind Metals (IWM)
- Official BattleTech miniatures: https://www.ironwindmetals.com/
- Track which mechs have available minis
- Stored in `model_sources` with `source_type = 'iwm'`

## Catalyst Force Packs
- Pre-packed miniature sets from Catalyst Game Labs
- Stored in `model_sources` with `source_type = 'forcepack'`
