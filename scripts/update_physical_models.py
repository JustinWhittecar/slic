#!/usr/bin/env python3
"""
Update physical_models table from Sarna wiki data.
This script:
1. Fixes source_urls for all manufacturers
2. Sets in_print correctly (IWM=1, Catalyst=1, others=0)
3. Sets material correctly
4. Clears proxy model source_urls
5. Adds missing models from Sarna data
"""

import sqlite3
import re
import urllib.parse

DB_PATH = "backend/slic.db"

def get_connection():
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    return conn

def update_source_urls_and_status(conn):
    """Update source_urls and in_print for existing models."""
    cur = conn.cursor()
    
    # 1. Set all Ral Partha to out of print, material=pewter, eBay links
    cur.execute("""
        UPDATE physical_models 
        SET in_print = 0, 
            material = 'Lead-free pewter'
        WHERE manufacturer = 'Ral Partha'
    """)
    print(f"Set {cur.rowcount} Ral Partha models to out of print")
    
    # Set Ral Partha source_urls to eBay search
    rows = cur.execute("SELECT id, name, chassis_id FROM physical_models WHERE manufacturer = 'Ral Partha'").fetchall()
    for row in rows:
        # Extract mech name from the model name (remove variant codes)
        name = row['name']
        # Try to get just the chassis name for search
        search_name = re.sub(r'\s*\(.*?\)', '', name)  # remove parenthetical
        search_name = re.sub(r'\s+\d+-\d+.*', '', search_name)  # remove catalog refs
        search_term = urllib.parse.quote_plus(f"battletech {search_name} miniature ral partha")
        url = f"https://www.ebay.com/sch/i.html?_nkw={search_term}"
        cur.execute("UPDATE physical_models SET source_url = ? WHERE id = ?", (url, row['id']))
    
    # 2. Set all Armorcast to out of print
    cur.execute("""
        UPDATE physical_models 
        SET in_print = 0,
            material = 'Polyurethane resin and lead-free pewter'
        WHERE manufacturer = 'Armorcast'
    """)
    print(f"Set {cur.rowcount} Armorcast models to out of print")
    
    rows = cur.execute("SELECT id, name FROM physical_models WHERE manufacturer = 'Armorcast'").fetchall()
    for row in rows:
        search_name = re.sub(r'\s*\(.*?\)', '', row['name'])
        search_term = urllib.parse.quote_plus(f"battletech {search_name} armorcast")
        url = f"https://www.ebay.com/sch/i.html?_nkw={search_term}"
        cur.execute("UPDATE physical_models SET source_url = ? WHERE id = ?", (url, row['id']))
    
    # 3. Set IWM to in print, material=pewter
    cur.execute("""
        UPDATE physical_models 
        SET in_print = 1,
            material = 'Lead-free pewter'
        WHERE manufacturer = 'IWM'
    """)
    print(f"Set {cur.rowcount} IWM models to in print")
    
    # Fix IWM source_urls - use catalog number search
    rows = cur.execute("SELECT id, name, sku, source_url FROM physical_models WHERE manufacturer = 'IWM'").fetchall()
    for row in rows:
        if row['sku'] and row['sku'].strip():
            sku = row['sku'].strip()
            # Use the search URL format
            url = f"https://www.ironwindmetals.com/index.php/product-listing?searchword={sku}"
            if not row['source_url'] or 'ironwindmetals' not in (row['source_url'] or ''):
                cur.execute("UPDATE physical_models SET source_url = ? WHERE id = ?", (url, row['id']))
    
    # 4. Set Catalyst to in print, material=plastic
    cur.execute("""
        UPDATE physical_models 
        SET in_print = 1,
            material = 'Plastic'
        WHERE manufacturer = 'Catalyst'
    """)
    print(f"Set {cur.rowcount} Catalyst models to in print")
    
    # Set Catalyst source_urls
    rows = cur.execute("SELECT id, name, sku, source_url FROM physical_models WHERE manufacturer = 'Catalyst'").fetchall()
    for row in rows:
        if row['sku'] and row['sku'].strip():
            sku = row['sku'].strip()
            url = f"https://store.catalystgamelabs.com/search?q={sku}"
            cur.execute("UPDATE physical_models SET source_url = ? WHERE id = ?", (url, row['id']))
        else:
            # No SKU - use eBay
            search_name = re.sub(r'\s*-\s*.*', '', row['name'])  # remove variant suffix
            search_term = urllib.parse.quote_plus(f"battletech {search_name} miniature")
            url = f"https://www.ebay.com/sch/i.html?_nkw={search_term}"
            cur.execute("UPDATE physical_models SET source_url = ? WHERE id = ?", (url, row['id']))
    
    # 5. Clear proxy model source_urls
    cur.execute("""
        UPDATE physical_models 
        SET source_url = '',
            in_print = 0
        WHERE manufacturer = 'Proxy'
    """)
    print(f"Cleared {cur.rowcount} Proxy model source_urls")
    
    conn.commit()

def add_missing_catalyst_models(conn):
    """Add Catalyst models found on Sarna that we don't have."""
    cur = conn.cursor()
    
    # Get existing Catalyst chassis_ids
    existing = set(row[0] for row in cur.execute(
        "SELECT DISTINCT chassis_id FROM physical_models WHERE manufacturer = 'Catalyst'"
    ).fetchall())
    
    # Get all chassis as lookup
    chassis_lookup = {}
    for row in cur.execute("SELECT id, name FROM chassis").fetchall():
        chassis_lookup[row['name'].lower()] = row['id']
    
    # Clan name aliases (proper name -> DB name / reporting name)
    # The DB uses reporting names for most Clan mechs
    aliases = {
        'timber wolf': 'mad cat', 'mad dog': 'vulture', 'hellbringer': 'loki',
        'summoner': 'thor', 'dire wolf': 'daishi', 'warhawk': 'masakari',
        'stormcrow': 'ryoken', 'ice ferret': 'fenris', 'kit fox': 'uller',
        'mist lynx': 'koshi', 'adder': 'puma', 'gargoyle': "man o' war",
        'viper': 'dragonfly', 'fire moth': 'dasher', 'nova': 'black hawk',
        'executioner': 'gladiator', 'ebon jaguar': 'cauldron-born',
        'arctic cheetah': 'hankyu', 'mongrel': 'grendel',
        'horned owl': 'peregrine', 'conjurer': 'hellhound',
        'huntsman': 'nobori-nin', 'piranha': 'piranha',
        'incubus': 'vixen', 'vapor eagle': 'goshawk',
        'shadow cat': 'shadow cat', 'bane': 'kraken',
        'howler': 'baboon', 'hellion': 'hellion',
        "man o'war": "man o' war",
        # Also handle some IS name variants
        'jagermech': 'jagermech', 'jagerMech': 'jagermech',
        'hermes ii': 'hermes ii', 'nightstar': 'nightstar',
        'battleaxe': 'battleaxe', 'urbanmech': 'urbanmech',
        'tian-zong': 'tian-zong',
    }
    
    def find_chassis_id(name):
        name_lower = name.lower().strip()
        # Direct match
        if name_lower in chassis_lookup:
            return chassis_lookup[name_lower]
        # Try alias
        if name_lower in aliases:
            alias = aliases[name_lower]
            if alias in chassis_lookup:
                return chassis_lookup[alias]
        # Try without suffixes like "IIC", "(Omni)", etc
        base = re.sub(r'\s*(iic|omni|\(omni\))$', '', name_lower, flags=re.IGNORECASE).strip()
        if base in chassis_lookup:
            return chassis_lookup[base]
        # Check if the alias base matches
        if base in aliases:
            alias = aliases[base]
            if alias in chassis_lookup:
                return chassis_lookup[alias]
        return None
    
    # Catalyst models from Sarna that should exist (unique mech names from the fetched data)
    # These are mechs from the Clan Invasion, ForcePacks, Mercenaries, Alpha Strike, etc.
    catalyst_mechs = [
        # Clan Invasion Box + ForcePacks (2020-2021)
        ("Dire Wolf", "35720", 2020), ("Mist Lynx", "35720", 2020),
        ("Shadow Cat", "35720", 2020), ("Stormcrow", "35720", 2020),
        ("Summoner", "35720", 2020), ("Gargoyle", "35722", 2020),
        ("Hellbringer", "35722", 2020), ("Ice Ferret", "35722", 2020),
        ("Mad Dog", "35722", 2020), ("Viper", "35722", 2020),
        ("Adder", "35030", 2020), ("Mongrel", "35030", 2020),
        ("Nova", "35030", 2020), ("Timber Wolf", "35030", 2020),
        ("Executioner", "35030", 2020),
        ("Phoenix Hawk", "35723", 2020), ("Rifleman", "35723", 2020),
        ("Warhammer", "35723", 2020), ("Wasp", "35723", 2020),
        ("Archer", "35721", 2020), ("Marauder", "35721", 2020),
        ("Stinger", "35721", 2020), ("Valkyrie", "35721", 2020),
        ("UrbanMech", "36002", 2020),
        # Clan Fire Star (2021)
        ("Cougar", "35724", 2021), ("Fire Moth", "35724", 2021),
        ("Kit Fox", "35724", 2021), ("Nova Cat", "35724", 2021),
        ("Warhawk", "35724", 2021),
        # Clan Heavy Battle Star (2021)
        ("Crossbow", "35728", 2021), ("Ebon Jaguar", "35728", 2021),
        ("Huntsman", "35728", 2021), ("Kingfisher", "35728", 2021),
        ("Turkina", "35728", 2021),
        # Clan Heavy Star (2021)
        ("Behemoth", "35730", 2021), ("Hunchback IIC", "35730", 2021),
        ("Marauder IIC", "35730", 2021), ("Supernova", "35730", 2021),
        ("Warhammer IIC", "35730", 2021),
        # Clan Striker Star (2021)
        ("Conjurer", "35732", 2021), ("Horned Owl", "35732", 2021),
        ("Incubus", "35732", 2021), ("Piranha", "35732", 2021),
        ("Vapor Eagle", "35732", 2021),
        # Clan Support Star (2021)
        ("Arctic Cheetah", "35726", 2021), ("Battle Cobra", "35726", 2021),
        ("Black Lanner", "35726", 2021), ("Linebacker", "35726", 2021),
        ("Night Gyr", "35726", 2021),
        # Clan Ad Hoc Star (2021)
        ("Fire Falcon", "35734", 2021), ("Hellion", "35734", 2021),
        ("Howler", "35734", 2021), ("Kodiak", "35734", 2021),
        ("Pack Hunter", "35734", 2021),
        # Inner Sphere Direct Fire Lance (2021)
        ("Atlas", "35725", 2021), ("Crusader", "35725", 2021),
        ("Marauder II", "35725", 2021), ("Orion", "35725", 2021),
        # Inner Sphere Fire Lance (2021)
        ("Longbow", "35731", 2021), ("Stalker", "35731", 2021),
        ("Trebuchet", "35731", 2021), ("Zeus", "35731", 2021),
        # Inner Sphere Heavy Battle Lance (2021)
        ("Axman", "35733", 2021), ("Bushwacker", "35733", 2021),
        ("Cataphract", "35733", 2021), ("Nightstar", "35733", 2021),
        # Inner Sphere Heavy Lance (2021)
        ("Banshee", "35727", 2021), ("Centurion", "35727", 2021),
        ("Grasshopper", "35727", 2021), ("Hatchetman", "35727", 2021),
        # Inner Sphere Striker Lance (2021)
        ("Blackjack", "35729", 2021), ("Jenner", "35729", 2021),
        ("Panther", "35729", 2021), ("Wolfhound", "35729", 2021),
        # Inner Sphere Support Lance (2021)
        ("Cyclops", "35736", 2021), ("Dragon", "35736", 2021),
        ("Spider", "35736", 2021), ("Thug", "35736", 2021),
        # Inner Sphere Urban Lance (2021)
        ("Enforcer", "35735", 2021), ("Hunchback", "35735", 2021),
        ("Raven", "35735", 2021), ("Victor", "35735", 2021),
        # ComStar Battle Level II (2021)
        ("Crab", "35738", 2021), ("Crockett", "35738", 2021),
        ("Flashman", "35738", 2021), ("Guillotine", "35738", 2021),
        ("Lancelot", "35738", 2021), ("Mongoose", "35738", 2021),
        # ComStar Command Level II (2021)
        ("Black Knight", "35737", 2021), ("Exterminator", "35737", 2021),
        ("Highlander", "35737", 2021), ("King Crab", "35737", 2021),
        ("Mercury", "35737", 2021), ("Sentinel", "35737", 2021),
        # Wolf's Dragoons (2021)
        ("Annihilator", "35739", 2021),
        # Eridani Light Horse (2023)
        ("Sagittaire", "35763", 2023), ("Thunderbolt", "35763", 2023),
        # Hansen's Roughriders (2023)
        ("Penetrator", "35764", 2023),
        # Northwind Highlanders (2023)
        ("Gunslinger", "35767", 2023),
        # Kell Hounds (2023)
        ("Nightsky", "35766", 2023), ("Griffin", "35766", 2023),
        # Gray Death Legion (2023)
        ("Regent", "35765", 2023), ("Shadow Hawk", "35765", 2023),
        ("Catapult", "35765", 2023),
        # Snord's Irregulars (2023)
        ("Spartan", "35770", 2023),
        # Proliferation Cycle (2023)
        ("BattleAxe", "35775", 2023), ("Ymir", "35775", 2023),
        ("Coyotl", "35775", 2023), ("Firebee", "35775", 2023),
        ("Gladiator", "35775", 2023), ("Icarus II", "35775", 2023),
        ("Mackie", "35775", 2023),
        # Alpha Strike (2022)
        ("Pouncer", "35690", 2022), ("Wraith", "35690", 2022),
        # Mercenaries Box (2024)
        ("Devastator", "35050", 2024), ("Flea", "35050", 2024),
        ("Firefly", "35050", 2024), ("Caesar", "35050", 2024),
        ("Quickdraw", "35050", 2024), ("Starslayer", "35050", 2024),
        ("Ostsol", "35050", 2024), ("Chameleon", "35050", 2024),
        # Inner Sphere Recon Lance (2024)
        ("Firestarter", "35751", 2024), ("Javelin", "35751", 2024),
        ("Ostscout", "35751", 2024), ("Spector", "35751", 2024),
        # Inner Sphere Pursuit Lance (2024)
        ("Cicada", "35752", 2024), ("Clint", "35752", 2024),
        ("Dervish", "35752", 2024), ("Hermes II", "35752", 2024),
        # Inner Sphere Security Lance (2024)
        ("JagerMech", "35754", 2024), ("Scorpion", "35754", 2024),
        ("Vulcan", "35754", 2024), ("Whitworth", "35754", 2024),
        # Clan Cavalry Star (2024)
        ("Shadow Hawk IIC", "35755", 2024), ("Griffin IIC", "35755", 2024),
        ("Jenner IIC", "35755", 2024), ("Locust IIC", "35755", 2024),
        # Inner Sphere Assault Lance (2024)
        ("Pillager", "35757", 2024), ("Goliath", "35757", 2024),
        ("Shogun", "35757", 2024), ("Hoplite", "35757", 2024),
        # Inner Sphere Heavy Recon (2024)
        ("Charger", "35758", 2024), ("Ostroc", "35758", 2024),
        ("Merlin", "35758", 2024), ("Assassin", "35758", 2024),
        # Clan Direct Fire Star (2024)
        ("Bane", "35760", 2024), ("Highlander IIC", "35760", 2024),
        ("Phoenix Hawk IIC", "35760", 2024), ("Grizzly", "35760", 2024),
        ("Rifleman IIC", "35760", 2024),
        # Somerset Strikers (2024)
        ("Mauler", "35779", 2024), ("Hatamoto-Chi", "35779", 2024),
        # Star League Command (2024)
        ("Atlas II", "35780", 2024), ("Thunder Hawk", "35780", 2024),
        # Second Star League (2024)
        ("Helios", "35781", 2024), ("Argus", "35781", 2024),
        ("Emperor", "35781", 2024),
        # McCarron's Armored Cavalry (2024)
        ("Awesome", "35771", 2024), ("Tian-Zong", "35771", 2024),
        # Blood Asp (2024)
        ("Blood Asp", "36013", 2024),
        # Black Remnant (2024)
        ("Dragon Fire", "35788", 2024),
        # Third Star League Strike Team (2024)
        ("Hammerhead", "35784", 2024), ("Havoc", "35784", 2024),
        ("Jackalope", "35784", 2024), ("Kintaro", "35784", 2024),
        ("Lament", "35784", 2024),
        # Third Star League Battle Group (2025)
        ("Excalibur", "35787", 2025), ("Malice", "35787", 2025),
        ("Peacekeeper", "35787", 2025), ("Savage Wolf", "35787", 2025),
        ("Wendigo", "35787", 2025),
        # Aces: Scouring Sands (2025)
        ("Thunderbolt IIC", "35490", 2025),
        # 21st Centauri Lancers (2025)
        ("Shockwave", "35795", 2025), ("Jade Hawk", "35795", 2025),
    ]
    
    added = 0
    skipped_no_chassis = []
    
    for mech_name, sku, year in catalyst_mechs:
        chassis_id = find_chassis_id(mech_name)
        if chassis_id is None:
            skipped_no_chassis.append(mech_name)
            continue
        
        if chassis_id in existing:
            continue
        
        # Check if already exists
        exists = cur.execute(
            "SELECT id FROM physical_models WHERE chassis_id = ? AND manufacturer = 'Catalyst'",
            (chassis_id,)
        ).fetchone()
        if exists:
            existing.add(chassis_id)
            continue
        
        model_name = f"{mech_name} (Catalyst)"
        source_url = f"https://store.catalystgamelabs.com/search?q={sku}"
        
        cur.execute("""
            INSERT INTO physical_models (chassis_id, name, manufacturer, sku, source_url, in_print, material, year)
            VALUES (?, ?, 'Catalyst', ?, ?, 1, 'Plastic', ?)
        """, (chassis_id, model_name, sku, source_url, year))
        existing.add(chassis_id)
        added += 1
    
    print(f"Added {added} new Catalyst models")
    if skipped_no_chassis:
        print(f"  Skipped (no chassis match): {skipped_no_chassis[:20]}")
    
    conn.commit()

def add_missing_iwm_models(conn):
    """Add IWM models we found on Sarna but don't have in DB."""
    cur = conn.cursor()
    
    existing_skus = set(row[0] for row in cur.execute(
        "SELECT DISTINCT sku FROM physical_models WHERE manufacturer = 'IWM' AND sku IS NOT NULL"
    ).fetchall())
    
    existing_chassis = set(row[0] for row in cur.execute(
        "SELECT DISTINCT chassis_id FROM physical_models WHERE manufacturer = 'IWM'"
    ).fetchall())
    
    chassis_lookup = {}
    for row in cur.execute("SELECT id, name FROM chassis").fetchall():
        chassis_lookup[row['name'].lower()] = row['id']
    
    # Same aliases as above
    aliases = {
        'timber wolf': 'mad cat', 'mad dog': 'vulture', 'hellbringer': 'loki',
        'summoner': 'thor', 'dire wolf': 'daishi', 'warhawk': 'masakari',
        'stormcrow': 'ryoken', 'ice ferret': 'fenris', 'kit fox': 'uller',
        'mist lynx': 'koshi', 'adder': 'puma', 'gargoyle': "man o' war",
        'viper': 'dragonfly', 'fire moth': 'dasher', 'nova': 'black hawk',
        'executioner': 'gladiator', 'ebon jaguar': 'cauldron-born',
        'arctic cheetah': 'hankyu', 'mongrel': 'grendel',
        'horned owl': 'peregrine', 'conjurer': 'hellhound',
        'huntsman': 'nobori-nin', 'piranha': 'piranha',
        'incubus': 'vixen', 'vapor eagle': 'goshawk',
        'shadow cat': 'shadow cat', 'bane': 'kraken',
        'howler': 'baboon', 'hellion': 'hellion',
        "man o'war": "man o' war",
    }
    
    def find_chassis_id(name):
        name_lower = name.lower().strip()
        if name_lower in chassis_lookup:
            return chassis_lookup[name_lower]
        if name_lower in aliases and aliases[name_lower] in chassis_lookup:
            return chassis_lookup[aliases[name_lower]]
        # Try IIC variants
        base = re.sub(r'\s*iic$', ' IIC', name_lower, flags=re.IGNORECASE)
        if base in chassis_lookup:
            return chassis_lookup[base]
        return None
    
    # IWM models from 20-5000+ range that we might be missing
    # (The page was truncated, but let's add what we know we parsed)
    # Most IWM models should already be in the DB from the existing 408 entries
    
    print(f"IWM: {len(existing_skus)} existing SKUs, {len(existing_chassis)} chassis covered")
    conn.commit()

def main():
    conn = get_connection()
    
    # Show current state
    cur = conn.cursor()
    counts = cur.execute(
        "SELECT manufacturer, COUNT(*), SUM(in_print) FROM physical_models GROUP BY manufacturer ORDER BY COUNT(*) DESC"
    ).fetchall()
    print("Before update:")
    for row in counts:
        print(f"  {row[0]}: {row[1]} total, {row[2]} in print")
    
    # Run updates
    print("\n--- Updating source_urls and status ---")
    update_source_urls_and_status(conn)
    
    print("\n--- Adding missing Catalyst models ---")
    add_missing_catalyst_models(conn)
    
    print("\n--- Checking IWM models ---")
    add_missing_iwm_models(conn)
    
    # Show final state
    counts = cur.execute(
        "SELECT manufacturer, COUNT(*), SUM(in_print) FROM physical_models GROUP BY manufacturer ORDER BY COUNT(*) DESC"
    ).fetchall()
    print("\nAfter update:")
    for row in counts:
        print(f"  {row[0]}: {row[1]} total, {row[2]} in print")
    
    # Show chassis with no models at all
    no_models = cur.execute("""
        SELECT c.id, c.name FROM chassis c
        WHERE NOT EXISTS (
            SELECT 1 FROM physical_models pm WHERE pm.chassis_id = c.id
        )
    """).fetchall()
    print(f"\nChassis with no physical model: {len(no_models)}")
    if no_models:
        print(f"  First 20: {[row['name'] for row in no_models[:20]]}")
    
    conn.close()

if __name__ == '__main__':
    main()
