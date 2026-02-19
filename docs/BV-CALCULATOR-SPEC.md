# BV2 Calculator Spec

## Goal
Calculate BV2 from first principles using MTF file data, verify against MegaMek published values. Store offensive/defensive BV split per variant for accurate pilot skill adjustments in the list builder.

## Why
- Published MUL BV uses a single multiplier for pilot skills (TM p.315 table)
- MegaMek actually splits into defensive BR + offensive BR
- Having the split lets us do **accurate** BV adjustments per the official formula
- Currently we use a simplified single-multiplier table normalized to G4/P5=1.0
- The "real" formula would apply gunnery modifier to offensive and piloting modifier to defensive separately (this is what MUL does)

## Data Sources
- **MTF files**: `data/megamek-data/data/mekfiles/meks/` — full mech specs
- **Equipment BV**: Already in `equipment` table (`bv` column) 
- **Ammo BV**: Need to add — not currently tracked in DB. Parse from MTF or MegaMek equipment files
- **TM tables**: Armor/Structure/Engine/Gyro modifiers, Defensive Factors, Speed Factors

## BV2 Formula (BattleMechs only, TM p.302-306)

### Step 1: Defensive Battle Rating (DBR)

```
armor_bv = total_armor × 2.5 × armor_type_modifier
structure_bv = total_IS × 1.5 × structure_type_modifier × engine_type_modifier  
gyro_bv = tonnage × gyro_modifier
defensive_equip_bv = sum(defensive equipment BV) + min(AMS_ammo_bv, AMS_weapon_bv)

subtotal = armor_bv + structure_bv + gyro_bv + defensive_equip_bv

# Subtract explosive penalties (cannot go below 1)
explosive_penalty = ammo_explosive_crits × 15 + gauss_crits × 1
subtotal = max(1, subtotal - explosive_penalty)

# Defensive Factor = 1 + (highest_TMM / 10)
# TMM based on best of run/jump with MASC/TSM bonus
DBR = subtotal × defensive_factor
```

#### Modifier Tables

**Armor Type**: Standard=1.0, Ferro-Fibrous=1.0, Industrial=1.0, Heavy Industrial=1.0, Commercial=0.5, Stealth=1.0

**Structure Type**: Standard=1.0, Endo-Steel=1.0, Industrial=0.5

**Engine Type** (for structure BV): Standard/ICE/FuelCell/Fission=1.0, Light=0.75, Compact=1.0, IS XL=0.5, Clan XL=0.75

**Gyro**: Standard=0.5, Compact=0.5, XL=0.5, Heavy-Duty=1.0

**Defensive Factor**: TMM+0=1.0, +1=1.1, +2=1.2, ..., +9+=1+(mod/10)

#### Explosive Ammo Penalties
- **Clan mechs** (assumed CASE): 15/crit in CT, legs, head only
- **IS XL engine**: 15/crit in ANY location
- **IS Standard/Light engine**: 15/crit in CT/legs/head OR non-CASE locations; 15/crit in arms not protected by CASE in location or next inward
- **Gauss weapons**: 1/crit following same location rules as ammo

### Step 2: Offensive Battle Rating (OBR)

```
# 1. Calculate each weapon's Modified BV
#    - Rear-facing weapons: ×0.5 (or front×0.5 if rear BV > front BV)
#    - Artemis IV: ×1.2, Artemis V: ×1.3
#    - Targeting Computer (direct fire): ×1.25
#    - Ammo BV capped at weapon BV (excessive ammo rule)

# 2. Heat efficiency
heat_efficiency = 6 + heat_sink_capacity - movement_heat
# movement_heat = max(running_heat, jumping_heat)
# running_heat = 2, jumping_heat = max(jump_mp, 3)
# Ultra AC: ×2 heat, RAC: ×6 heat, Streak SRM: ×0.5 heat, OS: ×0.25 heat

# 3. Weapon Battle Rating
# Sort weapons by Modified BV (desc), then heat (asc)
# Add weapons until heat_efficiency exceeded
# Weapon that exceeds = full BV, remaining = half BV
# Add ammo BV, equipment BV, tonnage (×1.5 if TSM)

# 4. Speed Factor
speed_mp = run_mp + ceil(jump_mp / 2)  # with MASC/TSM max
speed_factor = round((1 + (speed_mp - 5) / 10)^1.2, 2)

OBR = weapon_battle_rating × speed_factor
# IndustrialMech without AFC: ×0.9
```

### Step 3: Final BV
```
base_bv = DBR + OBR
# Small cockpit: ×0.95
final_bv = round(base_bv)  # .5 rounds up
```

## Implementation Plan

### Phase 1: MTF Parser Enhancement
- Parse ammo from MTF location blocks (count crits of "IS Ammo X" / "Clan Ammo X")
- Parse CASE presence per location
- Parse rear-mounted weapons
- Store: `variant_ammo` table or expand variant_equipment

### Phase 2: BV Calculator (Go)
- New package: `backend/internal/bvcalc/`
- `bvcalc.go`: Main calculator
- `tables.go`: All modifier tables from TM
- `ammo.go`: Ammo BV lookup and matching
- Input: variant data from DB + parsed MTF data
- Output: `{baseBV, defensiveBR, offensiveBR, details}`

### Phase 3: Verification
- CLI tool: `backend/cmd/verify-bv/main.go`
- Compare calculated BV vs published MUL BV for all 4,227 variants
- Report: exact matches, within 1%, within 5%, outliers
- Target: >95% within 1% of MegaMek values

### Phase 4: Store & Use
- Add columns to `variant_stats`: `calculated_bv`, `defensive_br`, `offensive_br`
- Update list builder to use split BV for pilot skill adjustments
- API: expose defensive/offensive split

## Ammo BV Reference (from TM p.317-318)
Need to build ammo BV lookup table. Key entries per ton:
- AC/2: 5, AC/5: 9, AC/10: 15, AC/20: 22
- LRM-5: 6, LRM-10: 11, LRM-15: 17, LRM-20: 23
- SRM-2: 3, SRM-4: 5, SRM-6: 7
- Gauss: 40§, Heavy Gauss: 43§, Light Gauss: 20§ (§ = non-explosive for ammo penalty purposes)
- LB-X same as equivalent AC
- Ultra same as equivalent AC  
- Streak SRM-2: 4, Streak SRM-4: 7, Streak SRM-6: 11
- MRM-10: 7, MRM-20: 14, MRM-30: 21, MRM-40: 28
- AMS: 11† (defensive)
- Machine Gun: 1

## Defensive Equipment (marked † in TM)
- AMS: 32
- A-Pod: 1
- B-Pod: 2
- Beagle Active Probe: 10
- Guardian ECM: 61
- Artemis IV: counted with weapon modifier, not separate
- Bridge-Layer: 5/10/20

## Notes
- MegaMek is authoritative — if our calc differs from MM, MM is right
- The split BV approach is more accurate for non-4/5 pilots than single multiplier
- We only need BattleMech BV calc (not vehicles, infantry, etc.)
- Stealth armor adds +10 to movement heat for BV purposes
- MASC: use max possible MP for speed factor
