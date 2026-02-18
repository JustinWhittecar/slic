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
| Atlas III AS7-D2 AS7-D2             |      4.0 |     18.0 |  10.00 |
| Atlas III AS7-D3 AS7-D3             |      4.0 |     14.5 |   9.51 |
| Atlas AS7-D (Ian) AS7-D (Ian)       |      5.0 |     15.0 |   8.85 |
| Atlas II AS7-D-H (Kerensky) AS7-D-H (Kerensky) |      5.0 |     13.0 |   8.34 |
| Atlas II AS7-DK-H AS7-DK-H          |      6.0 |     15.0 |   8.21 |
| Atlas II AS7-D-H2 AS7-D-H2          |      5.0 |     11.0 |   7.76 |
| Atlas AS7-Dr AS7-Dr                 |      6.0 |     13.0 |   7.71 |
| Atlas AS7-D-DC AS7-D-DC             |      6.0 |     12.0 |   7.43 |
| Atlas AS7-D (Danielle) AS7-D (Danielle) |      6.0 |     12.0 |   7.43 |
| Atlas II AS7-D-H AS7-D-H            |      6.0 |     12.0 |   7.43 |
| Atlas AS7-D AS7-D                   |      6.0 |     12.0 |   7.43 |
| Atlas II AS7-D-HT AS7-D-HT          |      6.0 |     12.0 |   7.43 |
| Atlas II AS7-D-H (Devlin) AS7-D-H (Devlin) |      7.0 |     13.0 |   7.17 |
| Hunchback HBK-4P HBK-4P             |      7.0 |      7.0 |   5.00 |

## Key Changes from V1
- Real 2D hex grid with terrain costs and LOS blocking
- Arc-based hit tables (front/rear) from actual positions
- Torso twist for weapon arc management
- Initiative determines movement order (second mover advantage)
- Woods cover (+1/+2 to-hit), elevation advantage (-1/+1)
- Terrain movement costs affect positioning
