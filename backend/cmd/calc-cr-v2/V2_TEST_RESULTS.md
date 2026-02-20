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
| Atlas III AS7-D2 AS7-D2             |      7.0 |     19.0 |   8.49 |
| Atlas AS7-D (Ian) AS7-D (Ian)       |      6.0 |     16.0 |   8.43 |
| Atlas AS7-K3 AS7-K3                 |      8.0 |     20.0 |   8.21 |
| Atlas AS7-S3 AS7-S3                 |      6.0 |     15.0 |   8.21 |
| Atlas AS8-S AS8-S                   |      6.0 |     14.0 |   7.97 |
| Atlas III AS7-D3 AS7-D3             |      6.0 |     14.0 |   7.97 |
| Atlas AS7-K2 AS7-K2                 |      7.0 |     16.0 |   7.89 |
| Atlas AS8-KE AS8-KE                 |      7.0 |     16.0 |   7.89 |
| Atlas AS7-K4 AS7-K4                 |      7.0 |     16.0 |   7.89 |
| Atlas AS7-S3-DC AS7-S3-DC           |      7.0 |     16.0 |   7.89 |
| Atlas AS7-K AS7-K                   |      7.0 |     15.5 |   7.78 |
| Atlas AS7-S (Hanssen) AS7-S (Hanssen) |      6.0 |     13.0 |   7.71 |
| Atlas C C                           |      6.0 |     13.0 |   7.71 |
| Atlas AS7-D-DC AS7-D-DC             |      6.0 |     13.0 |   7.71 |
| Atlas II AS7-D-H2 AS7-D-H2          |      6.0 |     13.0 |   7.71 |
| Atlas AS7-S4 AS7-S4                 |      7.0 |     15.0 |   7.67 |
| Atlas II AS7-DK-H AS7-DK-H          |      7.0 |     15.0 |   7.67 |
| Atlas C 3 C 3                       |      7.0 |     15.0 |   7.67 |
| Atlas AS7-00 (Jurn) AS7-00 (Jurn)   |      8.0 |     17.0 |   7.64 |
| Atlas AS8-K AS8-K                   |      7.0 |     14.0 |   7.43 |
| Atlas AS7-A AS7-A                   |     10.0 |     20.0 |   7.43 |
| Atlas C 2 C 2                       |      7.5 |     15.0 |   7.43 |
| Atlas AS7-D (Danielle) AS7-D (Danielle) |      6.0 |     12.0 |   7.43 |
| Atlas AS8-D AS8-D                   |      8.0 |     15.5 |   7.31 |
| Atlas AS7-S2 AS7-S2                 |     10.0 |     19.0 |   7.25 |
| Atlas AS7-K2 (Jedra) AS7-K2 (Jedra) |      8.0 |     15.0 |   7.20 |
| Atlas II AS7-D-HT AS7-D-HT          |      7.0 |     13.0 |   7.17 |
| Atlas AS7-K-DC AS7-K-DC             |      7.0 |     13.0 |   7.17 |
| Atlas AS7-Dr AS7-Dr                 |      7.0 |     13.0 |   7.17 |
| Atlas II AS7-D-H (Kerensky) AS7-D-H (Kerensky) |      7.0 |     13.0 |   7.17 |
| Atlas AS7-C AS7-C                   |      7.0 |     13.0 |   7.17 |
| Atlas AS7-CM AS7-CM                 |      7.0 |     13.0 |   7.17 |
| Atlas II AS7-D-H AS7-D-H            |      7.0 |     12.5 |   7.03 |
| Atlas AS7-RS AS7-RS                 |      8.0 |     14.0 |   6.96 |
| Atlas AS7-H AS7-H                   |      8.0 |     14.0 |   6.96 |
| Atlas AS7-WGS (Samsonov) AS7-WGS (Samsonov) |      8.0 |     14.0 |   6.96 |
| Atlas AS7-D AS7-D                   |      7.0 |     12.0 |   6.89 |
| Atlas II AS7-D-H (Devlin) AS7-D-H (Devlin) |      7.0 |     12.0 |   6.89 |
| Atlas AS7-S AS7-S                   |      6.5 |     11.0 |   6.84 |

## Key Changes from V1
- Real 2D hex grid with terrain costs and LOS blocking
- Arc-based hit tables (front/rear) from actual positions
- Torso twist for weapon arc management
- Initiative determines movement order (second mover advantage)
- Woods cover (+1/+2 to-hit), elevation advantage (-1/+1)
- Terrain movement costs affect positioning
