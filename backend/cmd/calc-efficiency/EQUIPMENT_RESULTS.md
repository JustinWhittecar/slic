# Equipment Implementation Results

## Date: 2026-02-17

## What Was Implemented

### 1. Targeting Computer
- Detects "Targeting Computer" in mech critical slots
- Applies −1 to-hit modifier to all direct-fire weapons (lasers, PPCs, ACs, Gauss, HAGs)
- Does NOT apply to missiles (LRM, SRM, MRM, ATM, Streak, MML) or artillery

### 2. Anti-Missile System (AMS)
- Detects "Anti-Missile System" / "AMS" in defender's critical slots
- On missile hit, reduces cluster hits by 1d6 (minimum 0)
- Standard AMS: 12 shots per ammo ton, consumed per activation
- Laser AMS: unlimited activations
- One AMS activation per turn (defends one salvo)

### 3. Special Ammo / Equipment

**a) LBX Cluster Mode**: Always fires cluster. −1 to-hit bonus applied. Each pellet = 1 damage to random location.

**b) Artemis IV**: Detected from slots. +2 to cluster table rolls for LRM, SRM, ATM, MML.

**c) Artemis V**: Detected from slots. +3 to cluster table rolls (same weapon types).

**d) Streak SRM**: Verified all-or-nothing behavior. AMS can still reduce hits post-hit-roll.

**e) Ultra AC**: Jam mechanic already implemented (natural 2 on second shot = jammed).

**f) Rotary AC**: Jam mechanic already implemented (natural 2 on any shot = jammed). Fires 6 shots.

## Test Results (selected)

| Mech | Offense | Defense | CR Score | Notes |
|------|---------|---------|----------|-------|
| HBK-4P | 6.0 | 6.0 | 5.00 | Baseline (no special equipment) ✓ |
| Mad Cat Prime | 5.0 | 13.0 | 8.34 | Artemis IV on LRMs + pulse lasers |
| Mad Cat Mk II (base) | 5.0 | 10.0 | 7.43 | Has TC |
| Mad Cat Mk II 4 | 6.0 | 17.0 | 8.65 | TC + heavy loadout |
| Mad Cat Mk II 5 | 5.0 | 14.0 | 8.60 | TC variant |
| Bushwacker BSW-X1 | 6.0 | 6.0 | 5.00 | LBX10 cluster mode |
| Daishi Prime | 5.0 | 10.0 | 7.43 | Heavy assault |
| Thor Prime | 6.0 | 13.0 | 7.71 | Mixed loadout |
| Vulture Prime | 5.0 | 10.0 | 7.43 | Missile boat |

## Fields Added to MechState
- `HasTargetingComputer bool`
- `HasAMS bool`, `AMSAmmo int`, `IsLaserAMS bool`, `AMSUsedThisTurn bool`
- `HasArtemisIV bool`, `HasArtemisV bool`

## Functions Added
- `clusterHitsWithBonus(rackSize, bonus int, rng)` - cluster table with Artemis modifier
- `isDirectFire(cat weaponCategory) bool` - TC eligibility check
- `amsIntercept(hits int, defender, rng) int` - AMS reduction
- `artemisBonus(m *MechState) int` - returns cluster bonus for attacker
