# SLIC Combat Rating V2 — Test Results

## Configuration
- Sims per board pair: 10
- Board pairs per mech: 20
- Total sims per mech: 200
- Max turns: 200
- Gunnery/Piloting: 4/5
- Board: 2x 16x17 standard boards combined (32x17)
- Deployment: attacker rows 1-3, defender rows 15-17
- 2D hex grid with terrain, LOS, arcs, torso twist

## Results

| Mech                                |  Offense |  Defense |     CR |
|-------------------------------------|----------|----------|--------|
| Archer ARC-2Rb ARC-2Rb              |     12.0 |     14.5 |   5.66 |
| Vixen 9 9                           |     10.0 |     12.0 |   5.64 |
| Archer ARC-2R ARC-2R                |     12.0 |     13.0 |   5.28 |
| Catapult CPLT-C1b CPLT-C1b          |     11.0 |     11.0 |   5.00 |
| Catapult CPLT-C1 (Jenny) 'Butterbee' CPLT-C1 (Jenny) 'Butterbee' |     11.0 |     10.0 |   4.67 |
| Star Python                         |     10.0 |      9.0 |   4.63 |
| Catapult CPLT-C1 CPLT-C1            |     13.0 |     10.0 |   4.08 |

## Key Changes from V1
- Real 2D hex grid with terrain costs and LOS blocking
- Arc-based hit tables (front/rear) from actual positions
- Torso twist for weapon arc management
- Initiative determines movement order (second mover advantage)
- Woods cover (+1/+2 to-hit), elevation advantage (-1/+1)
- Terrain movement costs affect positioning
