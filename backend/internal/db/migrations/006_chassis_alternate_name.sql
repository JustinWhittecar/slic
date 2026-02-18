ALTER TABLE chassis ADD COLUMN alternate_name TEXT;

-- IS reporting name → Clan name (or vice versa)
UPDATE chassis SET alternate_name = 'Timber Wolf' WHERE name = 'Mad Cat';
UPDATE chassis SET alternate_name = CASE name
    WHEN 'Daishi' THEN 'Dire Wolf'
    WHEN 'Thor' THEN 'Summoner'
    WHEN 'Loki' THEN 'Hellbringer'
    WHEN 'Ryoken' THEN 'Stormcrow'
    WHEN 'Vulture' THEN 'Mad Dog'
    WHEN 'Black Hawk' THEN 'Nova'
    WHEN 'Puma' THEN 'Adder'
    WHEN 'Uller' THEN 'Kit Fox'
    WHEN 'Masakari' THEN 'Warhawk'
    WHEN 'Fenris' THEN 'Ice Ferret'
    WHEN 'Koshi' THEN 'Mist Lynx'
    WHEN 'Dasher' THEN 'Fire Moth'
    WHEN 'Gladiator' THEN 'Executioner'
    WHEN 'Cauldron-Born' THEN 'Ebon Jaguar'
    WHEN 'Nobori-nin' THEN 'Huntsman'
    WHEN 'Hankyu' THEN 'Arctic Cheetah'
END
WHERE name IN ('Daishi','Thor','Loki','Ryoken','Vulture','Black Hawk','Puma','Uller','Masakari','Fenris','Koshi','Dasher','Gladiator','Cauldron-Born','Nobori-nin','Hankyu');

-- Viper/Dragonfly are both Clan mechs with swapped names
UPDATE chassis SET alternate_name = 'Dragonfly' WHERE name = 'Viper' AND tech_base = 'Clan';
UPDATE chassis SET alternate_name = 'Viper' WHERE name = 'Dragonfly' AND tech_base = 'Clan';
