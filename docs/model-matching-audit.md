# Physical Model-Chassis Matching Audit

**Date**: 2026-02-19  
**Branch**: `fix/model-matching`

## Summary

All 1,622 physical models are correctly matched to chassis. No mismatches found.

## Stats

| Metric | Count |
|--------|-------|
| Total physical models | 1,622 |
| Matched to chassis | 1,622 (100%) |
| Chassis in DB | 732 |
| Variants in DB | 4,227 |

### By Manufacturer
- Proxy: 732 (1 per chassis, auto-generated)
- IWM: 408
- Ral Partha: 274
- Catalyst: 205
- Armorcast: 3

## Spelling Variations (All Correct)

9 models have name differences from their chassis — all verified correct:
- Wardog → War Dog, Wolftrap → Wolf Trap (Tora), Blackwatch → Black Watch
- O'Bakemono → O-Bakemono, Rajin → Raijin, Menshen → Men Shen
- No-Daichi → No-Dachi, Man O War → Man O' War, Wolf Trap "Tora" → Wolf Trap (Tora)

## Top Proxy-Only Chassis (No Real Miniatures)

These are market gaps, not matching failures:
Crosscut (18 variants), Carbine (13), Grand Dragon (12), Brigand (11), Star Adder (10), Grigori (10)

## Fixes Applied

1. **SQLite schema**: Added missing `material` and `year` columns to `physical_models`
2. **Models API handler**: Added Material/Year fields to struct and query
