# PSR (Piloting Skill Rolls) & Falling Implementation Results

## Date: 2026-02-17

## What Was Implemented

### PSR Triggers
- **20+ potential damage in a single phase**: PSR with +1 modifier
- **Kicked (physical attack)**: PSR with +0 modifier
- **Leg/foot actuator destroyed**: +1 preexisting modifier per actuator
- **Hip actuator destroyed**: +2 preexisting modifier (replaces leg/foot mods for that leg)
- **Gyro hit**: +3 preexisting modifier per hit
- **Gyro destroyed (2 hits)**: Automatic fall (already treated as mech destroyed in existing code)

### PSR Resolution
- Roll 2d6 >= pilotingSkill (5) + preexisting modifiers + situational modifier
- Prone mechs ignore PSRs (except when standing)
- Unconscious pilot auto-fails all PSRs

### Falling
- Mech becomes prone
- Falling damage: tonnage/10 (round up), applied in 5-point groups to random locations
- Facing after fall: 1d6 table (1=Front, 2-3=Right, 4=Rear, 5-6=Left)
- Pilot damage check after fall (PSR, fail = +1 pilot damage)
- Consciousness check: thresholds 3/5/7/10/11 for 1-5 hits, 6 = dead

### Prone State
- Cannot move; attempts to stand each turn (costs 1 heat, requires PSR)
- Failed stand = falls again (takes falling damage again)
- Prone attacker: +1 to-hit modifier on all weapons
- Shooting at prone target: +1 at range ≤1, -2 at range >1

### Integration
- PSR checked after weapon fire phase if total fired damage >= 20
- PSR checked after critical hits to gyro/actuators (NeedsPSRFromCrit flag)
- PSR checked after physical attack (kick)
- Prone handling at start of turn (stand attempt before movement)
- Shutdown → fall (existing mechanic improved to use applyFall)

## Combat Rating Results (with PSR/Falling)

| Mech | Model | Offense (turns) | Defense (turns) | CR Score | Notes |
|------|-------|-----------------|-----------------|----------|-------|
| Hunchback | HBK-4P | 6.0 | 6.0 | 5.00 | Baseline (as expected) |
| Atlas | AS7-D | 5.0 | 9.0 | 7.06 | Heavy armor, rarely falls |
| Locust | LCT-1V | 12.0 | 3.0 | 1.00 | Light armor, death spiral from falls |
| Hatchetman | HCT-3F | 7.0 | 5.0 | 3.82 | Kick PSR interaction working |
| Daishi (Dire Wolf) | Prime | 5.0 | 10.0 | 7.43 | High alpha forces PSRs on opponents |

## Analysis

- **HBK-4P**: Stays at 5.0 as expected (it's the baseline)
- **Atlas AS7-D**: 7.06 — high armor means it rarely takes enough damage for PSR triggers. Falling is rare.
- **LCT-1V**: 1.00 (floor) — 20-ton mech gets devastated by falls. Even 2 damage from fall (20/10=2) is significant on light armor. The death spiral (prone → can't move → easy target → more damage → more PSRs) is working.
- **HCT-3F**: 3.82 — moderate. Physical attacks triggering defender PSRs work correctly.
- **Daishi Prime**: 7.43 — high alpha damage forces 20+ damage PSRs on opponents, amplifying its offensive power.

## Key Design Decisions

1. **20+ damage check uses total potential damage** (not actual hits) as a simplified approximation. This slightly over-triggers PSRs but avoids tracking per-weapon hit resolution damage.
2. **Prone mechs ignore all PSRs** except standing attempts (per BattleTech rules).
3. **0-level falls only** — flat terrain sim, no multi-level falls.
4. **NeedsPSRFromCrit flag** — set during critical hit resolution, checked after all damage for that weapon fire phase resolves. Prevents mid-resolution PSR cascades.
5. **Changes isolated from heat/weapon selection** — PSR code is independent of the EV-based heat model being developed concurrently.
