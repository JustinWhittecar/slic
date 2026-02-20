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
| Atlas III AS7-D2 AS7-D2             |      6.0 |     19.5 |   9.13 |
| Atlas AS7-D (Ian) AS7-D (Ian)       |      6.0 |     16.0 |   8.43 |
| Atlas III AS7-D3 AS7-D3             |      6.0 |     15.0 |   8.21 |
| Atlas II AS7-DK-H AS7-DK-H          |      6.0 |     15.0 |   8.21 |
| Atlas II AS7-D-H2 AS7-D-H2          |      6.0 |     14.0 |   7.97 |
| Atlas AS7-D AS7-D                   |      6.0 |     12.0 |   7.43 |
| Atlas AS7-Dr AS7-Dr                 |      7.0 |     14.0 |   7.43 |
| Atlas AS7-D-DC AS7-D-DC             |      7.0 |     14.0 |   7.43 |
| Atlas II AS7-D-H AS7-D-H            |      7.0 |     14.0 |   7.43 |
| Atlas II AS7-D-H (Kerensky) AS7-D-H (Kerensky) |      7.0 |     14.0 |   7.43 |
| Atlas II AS7-D-HT AS7-D-HT          |      7.0 |     14.0 |   7.43 |
| Atlas II AS7-D-H (Devlin) AS7-D-H (Devlin) |      7.0 |     13.0 |   7.17 |
| Atlas AS7-D (Danielle) AS7-D (Danielle) |      7.0 |     11.0 |   6.58 |
| Hunchback HBK-4P HBK-4P             |      8.0 |      8.0 |   5.00 |
| Marauder MAD-3R MAD-3R              |     11.0 |     10.0 |   4.67 |
| Locust LCT-1Vb LCT-1Vb              |     11.0 |      6.0 |   2.88 |
| Locust LCT-1V LCT-1V                |     71.0 |     14.0 |   1.00 |
| Locust LCT-1V2 LCT-1V2              |     77.0 |     15.0 |   1.00 |

## Key Changes from V1
- Real 2D hex grid with terrain costs and LOS blocking
- Arc-based hit tables (front/rear) from actual positions
- Torso twist for weapon arc management
- Initiative determines movement order (second mover advantage)
- Woods cover (+1/+2 to-hit), elevation advantage (-1/+1)
- Terrain movement costs affect positioning
