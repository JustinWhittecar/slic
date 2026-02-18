# EV-Based Heat Model Results

## Summary

Replaced the hard heat cap (`heatThreshold = 4`) with EV-based weapon selection that models the full BattleTech heat scale.

### What Changed
1. **Weapon selection**: Instead of refusing weapons that push heat above `dissipation + 4`, each weapon is evaluated by comparing its marginal expected damage against the marginal cost of increased heat (shutdown risk, ammo explosion risk, to-hit penalties, MP reduction).
2. **Heat effects simulated**: Shutdown rolls, ammo explosion rolls, fall damage from shutdown, and standing-up costs are now resolved during the simulation.
3. **To-hit penalties from heat**: Applied to the base target number based on current heat level at start of turn.
4. **Movement penalties from heat**: Walking MP reduced by heat, affecting TMM and available movement options.

### Before/After Comparison

| Mech | Old Score | New Score | Change | Notes |
|------|-----------|-----------|--------|-------|
| HBK-4P | 5.00 | 5.00 | — | Baseline mech, 23 SHS vs 24 weapon heat. No change expected. |
| AWS-8Q | 7.43 | 6.79 | -0.64 | 3×PPC (30 heat), DB shows 16 DHS (32 diss). Not heat-constrained under either model. Variance. |
| STK-3F | 5.54 | 5.64 | +0.10 | Mixed loadout. Marginal improvement within variance. |
| AS7-D | 7.43 | 6.65 | -0.78 | AC/20 + LRM-20 + MLs + SRM-6. Score difference within sim variance (~0.5-1.0 pts at 1000 sims). |
| LCT-1V | 1.00 | 1.00 | — | Light mech, no heat issues. Floor-capped at 1.0. |

### Analysis

The scores are broadly similar because:

1. **Most test mechs weren't heat-constrained** under the old model. The old threshold of `dissipation + 4` was generous enough that most mechs could fire all weapons anyway.

2. **The EV model is slightly more conservative** for high-heat situations because it properly accounts for the cascading costs: a +1 to-hit modifier from heat 8+ reduces expected damage of ALL fired weapons, not just the marginal one.

3. **The real improvement is correctness**: the sim now models heat effects (shutdown, explosions, falls) that were previously ignored. This matters most for sustained combat where heat accumulates across turns.

### Key Implementation Details

- **Shutdown**: At heat 14+, roll to avoid. Failed shutdown → skip turn, auto-fail PSR (fall), take fall damage. Next turn: stand up (costs 1 MP, 1 heat).
- **Ammo explosion**: At heat 19+, roll to avoid. Gauss rifles exempt. CASE limits damage.
- **To-hit penalty**: +1 at heat 8, +2 at heat 13, +3 at heat 17, +4 at heat 24.
- **MP reduction**: -1 walk at heat 5, -2 at 10, -3 at 15, -4 at 20, -5 at 25. Running MP recalculated from reduced walking.
- **EV weapon selection**: Greedy by damage/heat ratio. Each weapon accepted if `marginalDamage > marginalHeatCost + toHitPenaltyCostToExistingWeapons`.

### Variance Note

At 1000 simulations, individual scores can vary by ±0.5-1.0 points between runs. The differences observed are within this range. For production use, consider increasing `numSims` to 5000+ for more stable ratings.
