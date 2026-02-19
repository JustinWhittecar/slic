-- WizKids / MechWarrior: Dark Age miniatures insert
-- Only BattleMechs (skip vehicles, infantry, battle armor)
-- One entry per unique mech sculpt per expansion

-- Helper: create temp table for batch matching
CREATE TEMP TABLE wizkids_staging (
    name TEXT NOT NULL,
    manufacturer TEXT NOT NULL DEFAULT 'WizKids',
    sku TEXT,
    material TEXT DEFAULT 'Prepainted Plastic',
    year INTEGER,
    source_url TEXT,
    chassis_match TEXT -- the chassis name to look up
);

-- ============================================================
-- MechWarrior: Dark Age (DA) - 2002
-- ============================================================
-- Non-Unique 'Mechs
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
('MiningMech (DA)', 'DA073', 2002, 'Burrower', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('ConstructionMech (DA)', 'DA075', 2002, 'Carbine', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('AgroMech (DA)', 'DA077', 2002, 'Harvester', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('ForestryMech (DA)', 'DA079', 2002, 'Crosscut', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('MiningMech MOD (DA)', 'DA081', 2002, 'Burrower', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('ConstructionMech MOD (DA)', 'DA084', 2002, 'Carbine', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('AgroMech MOD (DA)', 'DA087', 2002, 'Harvester', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('ForestryMech MOD (DA)', 'DA090', 2002, 'Crosscut', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Koshi (DA)', 'DA093', 2002, 'Koshi', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Spider (DA)', 'DA096', 2002, 'Spider', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Centurion (DA)', 'DA099', 2002, 'Centurion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Black Hawk (DA)', 'DA102', 2002, 'Black Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age');
-- Unique 'Mechs
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
('Arbalest (DA)', 'DA105', 2002, 'Arbalest', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Firestarter (DA)', 'DA106', 2002, 'Firestarter', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Panther (DA)', 'DA107', 2002, 'Panther', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Pack Hunter (DA)', 'DA108', 2002, 'Pack Hunter II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Hatchetman (DA)', 'DA109', 2002, 'Hatchetman', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Ryoken II (DA)', 'DA110', 2002, 'Ryoken II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Legionnaire (DA)', 'DA111', 2002, 'Legionnaire', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Catapult (DA)', 'DA112', 2002, 'Catapult', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Mad Cat III (DA)', 'DA113', 2002, 'Mad Cat III', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Tundra Wolf (DA)', 'DA114', 2002, 'Tundra Wolf', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Atlas (DA)', 'DA115', 2002, 'Atlas', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Jupiter (DA)', 'DA116', 2002, 'Jupiter', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age');
-- LE Mechs (unique sculpts that are mechs)
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
('Cougar (DA LE)', 'DA157', 2002, 'Cougar', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age'),
('Zeus (DA LE)', 'DA158', 2002, 'Zeus-X', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Dark_Age');

-- ============================================================
-- MechWarrior: Fire for Effect (FE) - 2003
-- ============================================================
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
-- Non-Unique (IndustrialMechs included)
('AgroMech Mk II (FE)', 'FE061', 2003, 'Demeter', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('AgroMech Mk II MOD (FE)', 'FE064', 2003, 'Demeter', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('AgroMech MOD-B (FE)', 'FE067', 2003, 'Harvester', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('MiningMech MOD-B (FE)', 'FE070', 2003, 'Ground Pounder', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Arbalest (FE)', 'FE073', 2003, 'Arbalest', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Firestarter (FE)', 'FE076', 2003, 'Firestarter', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Legionnaire (FE)', 'FE079', 2003, 'Legionnaire', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Hatchetman (FE)', 'FE082', 2003, 'Hatchetman', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
-- Unique 'Mechs
('Cougar (FE)', 'FE085', 2003, 'Cougar', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Uller (FE)', 'FE086', 2003, 'Uller', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Crimson Hawk (FE)', 'FE087', 2003, 'Crimson Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Centurion (FE)', 'FE088', 2003, 'Centurion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Shadow Cat II (FE)', 'FE089', 2003, 'Shadow Cat II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Black Hawk (FE)', 'FE090', 2003, 'Black Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Mad Cat Mk II (FE)', 'FE091', 2003, 'Mad Cat Mk II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Thor (FE)', 'FE092', 2003, 'Thor II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Vulture (FE)', 'FE093', 2003, 'Vulture', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Black Knight (FE)', 'FE094', 2003, 'Black Knight', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Zeus (FE)', 'FE095', 2003, 'Zeus-X', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect'),
('Cygnus (FE)', 'FE096', 2003, 'Cygnus', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Fire_for_Effect');

-- ============================================================
-- MechWarrior: Death From Above (DF) - 2004
-- ============================================================
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
-- Non-Unique
('ConstructionMech MOD-B (DFA)', 'DF081', 2004, 'Carbine', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('ConstructionMech Mk II (DFA)', 'DF085', 2004, 'Carbine', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('ConstructionMech MkII MOD (DFA)', 'DF089', 2004, 'Carbine', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Crimson Hawk (DFA)', 'DF093', 2004, 'Crimson Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Locust (DFA)', 'DF097', 2004, 'Locust', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Cougar (DFA)', 'DF101', 2004, 'Cougar', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Uller (DFA)', 'DF105', 2004, 'Uller', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Ghost (DFA)', 'DF109', 2004, 'Ghost', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
-- Unique 'Mechs
('Valiant (DFA)', 'DF113', 2004, 'Valiant', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Mongoose II (DFA)', 'DF114', 2004, 'Mongoose II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Ocelot (DFA)', 'DF115', 2004, 'Ocelot', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Wolfhound (DFA)', 'DF116', 2004, 'Wolfhound', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Sun Cobra (DFA)', 'DF117', 2004, 'Sun Cobra', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Shockwave (DFA)', 'DF118', 2004, 'Shockwave', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Thunderbolt (DFA)', 'DF119', 2004, 'Thunderbolt', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Sphinx (DFA)', 'DF120', 2004, 'Sphinx', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Black Knight (DFA)', 'DF121', 2004, 'Black Knight', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Mad Cat Mk II (DFA)', 'DF122', 2004, 'Mad Cat Mk II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Zeus (DFA)', 'DF123', 2004, 'Zeus-X', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above'),
('Atlas (DFA)', 'DF124', 2004, 'Atlas', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Death_From_Above');

-- ============================================================
-- MechWarrior: Counterassault (CA) - 2004
-- ============================================================
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
-- Non-Unique
('ForestryMech MOD-B (CA)', 'CA081', 2004, 'Crosscut', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Raider (CA)', 'CA085', 2004, 'Raider', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Raider MkII (CA)', 'CA089', 2004, 'Raider Mk II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Mjolnir (CA)', 'CA093', 2004, 'Mjolnir', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Phoenix Hawk (CA)', 'CA097', 2004, 'Phoenix Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Sun Cobra (CA)', 'CA101', 2004, 'Sun Cobra', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Shockwave (CA)', 'CA105', 2004, 'Shockwave', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Rifleman (CA)', 'CA109', 2004, 'Rifleman', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
-- Unique 'Mechs
('Locust (CA)', 'CA113', 2004, 'Locust', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Blade (CA)', 'CA114', 2004, 'Blade', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Wasp (CA)', 'CA115', 2004, 'Wasp', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Centurion (CA)', 'CA116', 2004, 'Centurion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Ghost (CA)', 'CA117', 2004, 'Ghost', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Ti Ts''ang (CA)', 'CA118', 2004, 'Ti Ts''ang', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Thunderbolt (CA)', 'CA119', 2004, 'Thunderbolt', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Sphinx (CA)', 'CA120', 2004, 'Sphinx', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Longbow (CA)', 'CA121', 2004, 'Longbow', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Yu Huang (CA)', 'CA122', 2004, 'Yu Huang', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Kodiak (CA)', 'CA123', 2004, 'Kodiak II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault'),
('Marauder II (CA)', 'CA124', 2004, 'Marauder II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Counterassault');

-- ============================================================
-- MechWarrior: Age of Destruction (AD) - 2005
-- ============================================================
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
-- Non-Unique
('Mongoose (AoD)', 'AD081', 2005, 'Mongoose II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Valiant (AoD)', 'AD085', 2005, 'Valiant', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Blade (AoD)', 'AD089', 2005, 'Blade', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Uziel (AoD)', 'AD093', 2005, 'Uziel', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Thunder Fox (AoD)', 'AD097', 2005, 'Thunder Fox', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Hatchetman (AoD)', 'AD101', 2005, 'Hatchetman', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Dasher II (AoD)', 'AD105', 2005, 'Dasher II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Ursa (AoD)', 'AD109', 2005, 'Ursa', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
-- Unique 'Mechs (Rare)
('Arbalest "Bolt" (AoD)', 'AD113', 2005, 'Arbalest', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Firestarter "Cinders" (AoD)', 'AD114', 2005, 'Firestarter', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Nyx "Kolyu" (AoD)', 'AD115', 2005, 'Nyx', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Cuirass "Crusader" (AoD)', 'AD116', 2005, 'Cuirass', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Mangonel "Alpha" (AoD)', 'AD117', 2005, 'Mangonel', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Rifleman "Big Bertha" (AoD)', 'AD118', 2005, 'Rifleman', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Jade Hawk "Milagro" (AoD)', 'AD119', 2005, 'Jade Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Cave Lion "Anima" (AoD)', 'AD120', 2005, 'Cave Lion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Phoenix Hawk IIC "Hellfire" (AoD)', 'AD121', 2005, 'Phoenix Hawk IIC', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('BattleMaster "Caber" (AoD)', 'AD122', 2005, 'BattleMaster', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Xanthos "Chikako" (AoD)', 'AD123', 2005, 'Xanthos', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Cygnus "Persuader" (AoD)', 'AD124', 2005, 'Cygnus', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
-- Starter Set mechs
('Jade Hawk "Blitz" (AoD Starter)', 'AD133', 2005, 'Jade Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Mangonel "Copperhead" (AoD Starter)', 'AD134', 2005, 'Mangonel', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
-- Super-Rare
('BattleMaster "E.O.D." (AoD SR)', 'AD135', 2005, 'BattleMaster', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Thunder Fox "Twinkletoes" (AoD SR)', 'AD136', 2005, 'Thunder Fox', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Ursa "Prowler" (AoD SR)', 'AD137', 2005, 'Ursa', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Xanthos "Wooly" (AoD SR)', 'AD138', 2005, 'Xanthos', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Jade Hawk "Phoenix" (AoD SR)', 'AD139', 2005, 'Jade Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
('Phoenix Hawk IIC "Fu" (AoD SR)', 'AD140', 2005, 'Phoenix Hawk IIC', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction'),
-- Ultra Rare
('Crimson Hawk (AoD UR)', 'AD141', 2005, 'Crimson Hawk', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Age_of_Destruction');

-- ============================================================
-- MechWarrior: Falcon''s Prey (FP) - 2006
-- ============================================================
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
-- Non-Unique
('SalvageMech MOD (FP)', 'FP081', 2006, 'Opossum', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Ocelot (FP)', 'FP085', 2006, 'Ocelot', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Panther (FP)', 'FP089', 2006, 'Panther', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Stinger (FP)', 'FP093', 2006, 'Stinger', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Spider (FP)', 'FP097', 2006, 'Spider', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Koshi (FP)', 'FP101', 2006, 'Koshi', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Shadow Hawk IIC (FP)', 'FP105', 2006, 'Shadow Hawk IIC', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Griffin (FP)', 'FP109', 2006, 'Griffin', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
-- Unique 'Mechs
('Uziel (FP)', 'FP113', 2006, 'Uziel', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Eyrie (FP)', 'FP114', 2006, 'Eyrie', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Storm Raider (FP)', 'FP115', 2006, 'Storm Raider', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Gyrfalcon (FP)', 'FP116', 2006, 'Gyrfalcon', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Tiburon (FP)', 'FP117', 2006, 'Tiburon', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Vulture (FP)', 'FP118', 2006, 'Vulture Mk III', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Thor (FP)', 'FP119', 2006, 'Thor II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Tundra Wolf (FP)', 'FP120', 2006, 'Tundra Wolf', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Warhammer IIC (FP)', 'FP121', 2006, 'Warhammer IIC', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Shrike (FP)', 'FP122', 2006, 'Shrike', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Cyclops (FP)', 'FP123', 2006, 'Cyclops', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey'),
('Templar (FP)', 'FP124', 2006, 'Templar', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Falcon%27s_Prey');

-- ============================================================
-- MechWarrior: Domination (DO) - 2006
-- ============================================================
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
-- Non-Unique
('Jackalope (DO)', 'DO081', 2006, 'Jackalope', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Phoenix Hawk I (DO)', 'DO085', 2006, 'Phoenix Hawk L ''Fenikkusu Taka''', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Mjolnir (DO)', 'DO089', 2006, 'Mjolnir', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Night Stalker (DO)', 'DO093', 2006, 'Night Stalker', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Violator (DO)', 'DO097', 2006, 'Violator', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Raptor II (DO)', 'DO101', 2006, 'Raptor II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Scourge (DO)', 'DO105', 2006, 'Scourge', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Loki (DO)', 'DO109', 2006, 'Hel', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
-- Unique 'Mechs
('Anubis "Haria" (DO)', 'DO113', 2006, 'Anubis', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Nyx "Fortune" (DO)', 'DO114', 2006, 'Nyx', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Nyx "Glory" (DO)', 'DO115', 2006, 'Nyx', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Solitaire "Diamond Jack" (DO)', 'DO116', 2006, 'Solitaire', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Enforcer III "Damocles" (DO)', 'DO117', 2006, 'Enforcer III', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Mongrel "Fang" (DO)', 'DO118', 2006, 'Mongrel', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Ursus "Ull" (DO)', 'DO119', 2006, 'Ursus', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Thunder Fox "Bartleby" (DO)', 'DO120', 2006, 'Thunder Fox', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Thor "Tremor" (DO)', 'DO121', 2006, 'Thor II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Karhu "Headhunter" (DO)', 'DO122', 2006, 'Karhu', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Mad Cat Mk IV "Alpha" (DO)', 'DO123', 2006, 'Mad Cat Mk IV', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Zeus "Mercy" (DO)', 'DO124', 2006, 'Zeus-X', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
-- Super-Rare
('Anubis "Spot" (DO SR)', 'DO125', 2006, 'Anubis', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Solitaire "Hermod" (DO SR)', 'DO126', 2006, 'Solitaire', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Mongrel "Hod" (DO SR)', 'DO127', 2006, 'Mongrel', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Ursus "Odin" (DO SR)', 'DO128', 2006, 'Ursus', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Thunder Fox "Frigga" (DO SR)', 'DO129', 2006, 'Thunder Fox', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Karhu "Balder" (DO SR)', 'DO130', 2006, 'Karhu', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Black Knight "Miss Direction" (DO SR)', 'DO131', 2006, 'Black Knight', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
-- LE
('Mad Cat Mk IV "The Heat" (DO LE)', 'DO132', 2006, 'Mad Cat Mk IV', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Jackalope "Thumper" (DO LE)', 'DO133', 2006, 'Jackalope', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Solitaire "Vengeance" (DO LE)', 'DO134', 2006, 'Solitaire', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination'),
('Jackalope "Harvey" (DO OC)', 'DOOC1', 2006, 'Jackalope', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Domination');

-- ============================================================
-- MechWarrior: Solaris VII (SOL) - 2007
-- ============================================================
INSERT INTO wizkids_staging (name, sku, year, chassis_match, source_url) VALUES
('Rokurokubi "Yojinbo" (SOL)', 'SOL001', 2007, 'Rokurokubi', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Hellion "Laser Bait" (SOL)', 'SOL002', 2007, 'Hellion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Koshi "Six Gun" (SOL)', 'SOL003', 2007, 'Koshi', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Spider "Lucky" (SOL)', 'SOL004', 2007, 'Spider', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Cadaver "Possum" (SOL)', 'SOL005', 2007, 'Cadaver', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Vixen "Valentine" (SOL)', 'SOL006', 2007, 'Vixen', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Nyx "Tick" (SOL)', 'SOL007', 2007, 'Nyx', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Solitaire "Metablade" (SOL)', 'SOL008', 2007, 'Solitaire', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Night Stalker "Rogue Wave" (SOL)', 'SOL009', 2007, 'Night Stalker', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Centurion "Maximus" (SOL)', 'SOL010', 2007, 'Centurion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Exhumer "Sweetness" (SOL)', 'SOL011', 2007, 'Exhumer', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Gravedigger "Iceberg" (SOL)', 'SOL012', 2007, 'Gravedigger', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Eisenfaust "Avalon" (SOL)', 'SOL013', 2007, 'Eisenfaust', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Violator "Gold Digger" (SOL)', 'SOL014', 2007, 'Violator', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Neanderthal "Silas" (SOL)', 'SOL015', 2007, 'Neanderthal', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Rifleman "Crucible" (SOL)', 'SOL016', 2007, 'Rifleman', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Cave Lion "Rosse''s Revenge" (SOL)', 'SOL017', 2007, 'Cave Lion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Warwolf "Black Tooth" (SOL)', 'SOL018', 2007, 'Warwolf', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Mortis "Seraph" (SOL)', 'SOL019', 2007, 'Mortis', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Catapult "Cynosure" (SOL)', 'SOL020', 2007, 'Catapult II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('BattleMaster "Linebacker" (SOL)', 'SOL021', 2007, 'BattleMaster', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Daishi "Corruptor" (SOL)', 'SOL022', 2007, 'Daishi', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Shrike "Celaeno" (SOL)', 'SOL023', 2007, 'Shrike', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Kodiak "Ogre" (SOL)', 'SOL024', 2007, 'Kodiak II', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
-- LE
('Cadaver "Red Sonja" (SOL LE)', 'SOL025', 2007, 'Cadaver', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Centurion "Yen-lo-Wang" (SOL LE)', 'SOL026', 2007, 'Centurion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Cave Lion "Sturmgreif" (SOL LE)', 'SOL027', 2007, 'Cave Lion', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII'),
('Daishi "Widowmaker" (SOL LE)', 'SOL028', 2007, 'Daishi', 'https://www.sarna.net/wiki/Miniatures_-_Wizkids/MechWarrior:_Solaris_VII');

-- ============================================================
-- FASA Miniatures
-- ============================================================
-- PlasTech (1988)
INSERT INTO wizkids_staging (name, manufacturer, sku, year, chassis_match, material, source_url) VALUES
('Atlas (PlasTech)', 'FASA', '1633', 1988, 'Atlas', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Catapult (PlasTech)', 'FASA', '1633', 1988, 'Catapult', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Hunchback (PlasTech)', 'FASA', '1633', 1988, 'Hunchback', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Trebuchet (PlasTech)', 'FASA', '1633', 1988, 'Trebuchet', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Blackjack (PlasTech)', 'FASA', '1633', 1988, 'Blackjack', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Panther (PlasTech)', 'FASA', '1633', 1988, 'Panther', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Valkyrie (PlasTech)', 'FASA', '1633', 1988, 'Valkyrie', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Locust (PlasTech)', 'FASA', '1633', 1988, 'Locust', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA');

-- BattleTech 3rd Edition (1992)
INSERT INTO wizkids_staging (name, manufacturer, sku, year, chassis_match, material, source_url) VALUES
('BattleMaster BLR-1G (BT3)', 'FASA', '1604', 1992, 'BattleMaster', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Warhammer WHM-6R (BT3)', 'FASA', '1604', 1992, 'Warhammer', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Marauder MAD-3R (BT3)', 'FASA', '1604', 1992, 'Marauder', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Archer ARC-2R (BT3)', 'FASA', '1604', 1992, 'Archer', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Crusader CRD-3R (BT3)', 'FASA', '1604', 1992, 'Crusader', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Rifleman RFL-3N (BT3)', 'FASA', '1604', 1992, 'Rifleman', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Thunderbolt TDR-5S (BT3)', 'FASA', '1604', 1992, 'Thunderbolt', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Griffin GRF-1N (BT3)', 'FASA', '1604', 1992, 'Griffin', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Shadow Hawk SHD-2H (BT3)', 'FASA', '1604', 1992, 'Shadow Hawk', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Wolverine WVR-6R (BT3)', 'FASA', '1604', 1992, 'Wolverine', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Phoenix Hawk PXH-1K (BT3)', 'FASA', '1604', 1992, 'Phoenix Hawk', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Wasp WSP-1A (BT3)', 'FASA', '1604', 1992, 'Wasp', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Stinger STG-3R (BT3)', 'FASA', '1604', 1992, 'Stinger', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Locust LCT-1V (BT3)', 'FASA', '1604', 1992, 'Locust', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA');

-- CityTech 2nd Edition (1994)
INSERT INTO wizkids_staging (name, manufacturer, sku, year, chassis_match, material, source_url) VALUES
('Javelin JVN-10P (CT2)', 'FASA', '1608', 1994, 'Javelin', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Centurion CN9-D (CT2)', 'FASA', '1608', 1994, 'Centurion', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Victor VTR-9K (CT2)', 'FASA', '1608', 1994, 'Victor', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Orion ON1-M (CT2)', 'FASA', '1608', 1994, 'Orion', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Uller (CT2)', 'FASA', '1608', 1994, 'Uller', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Black Hawk (CT2)', 'FASA', '1608', 1994, 'Black Hawk', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Mad Cat (CT2)', 'FASA', '1608', 1994, 'Mad Cat', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA'),
('Daishi (CT2)', 'FASA', '1608', 1994, 'Daishi', 'Plastic', 'https://www.sarna.net/wiki/Miniatures_-_FASA');

-- ============================================================
-- Now insert into physical_models from staging, matching chassis
-- ============================================================
INSERT INTO physical_models (name, manufacturer, sku, material, year, source_url, chassis_id, in_print)
SELECT 
    s.name,
    s.manufacturer,
    s.sku,
    s.material,
    s.year,
    s.source_url,
    c.id as chassis_id,
    false as in_print
FROM wizkids_staging s
LEFT JOIN chassis c ON c.name = s.chassis_match
WHERE NOT EXISTS (
    SELECT 1 FROM physical_models pm 
    WHERE pm.name = s.name AND pm.manufacturer = s.manufacturer
);

-- Report results
SELECT 'WizKids models added' as metric, COUNT(*) as count FROM physical_models WHERE manufacturer = 'WizKids';
SELECT 'FASA models added' as metric, COUNT(*) as count FROM physical_models WHERE manufacturer = 'FASA';
SELECT 'Matched to chassis' as metric, COUNT(*) as count FROM physical_models WHERE manufacturer IN ('WizKids', 'FASA') AND chassis_id IS NOT NULL;
SELECT 'Unmatched (no chassis)' as metric, COUNT(*) as count FROM physical_models WHERE manufacturer IN ('WizKids', 'FASA') AND chassis_id IS NULL;
SELECT name, sku, chassis_id FROM physical_models WHERE manufacturer IN ('WizKids', 'FASA') AND chassis_id IS NULL ORDER BY name;

DROP TABLE wizkids_staging;
