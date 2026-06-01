INSERT INTO planet.player_progress (user_id, vip_points, total_resources_produced)
SELECT p.user_id, 0, 0
FROM planet.planets p
WHERE NOT EXISTS (
    SELECT 1 FROM planet.player_progress pp
    WHERE pp.user_id = p.user_id
);

INSERT INTO planet.player_technologies (user_id, type, level)
SELECT p.user_id, 'energy_tech', 3
FROM planet.planets p
WHERE NOT EXISTS (
    SELECT 1 FROM planet.player_technologies t
    WHERE t.user_id = p.user_id AND t.type = 'energy_tech'
);

INSERT INTO planet.buildings (planet_id, type, level)
SELECT p.id, btype, 1
FROM planet.planets p
CROSS JOIN (VALUES ('robotics_factory'), ('nanite_factory'), ('terraformer'), ('fusion_reactor'), ('shipyard')) AS t(btype)
WHERE NOT EXISTS (
    SELECT 1 FROM planet.buildings b
    WHERE b.planet_id = p.id AND b.type = t.btype
);

INSERT INTO planet.buildings (planet_id, type, level)
SELECT p.id, btype, 0
FROM planet.planets p
CROSS JOIN (VALUES ('small_shield_dome'), ('large_shield_dome')) AS t(btype)
WHERE NOT EXISTS (
    SELECT 1 FROM planet.buildings b
    WHERE b.planet_id = p.id AND b.type = t.btype
);

INSERT INTO planet.buildings (planet_id, type, level)
SELECT p.id, 'missile_silo', 1
FROM planet.planets p
WHERE NOT EXISTS (
    SELECT 1 FROM planet.buildings b
    WHERE b.planet_id = p.id AND b.type = 'missile_silo'
);

INSERT INTO planet.player_ships (planet_id, ship_type, quantity)
SELECT p.id, s.ship_type, 0
FROM planet.planets p
CROSS JOIN (VALUES ('cargo'), ('large_cargo'), ('recycler'), ('espionage_probe'), ('colony_ship'), ('solar_satellite'), ('light_fighter'), ('heavy_fighter'), ('cruiser'), ('battleship'), ('dreadnought'), ('bomber')) AS s(ship_type)
WHERE NOT EXISTS (
    SELECT 1 FROM planet.player_ships ps
    WHERE ps.planet_id = p.id AND ps.ship_type = s.ship_type
);

INSERT INTO planet.player_defenses (planet_id, defense_type, quantity)
SELECT p.id, d.defense_type, 0
FROM planet.planets p
CROSS JOIN (VALUES ('rocket_launcher'), ('light_laser'), ('heavy_laser'), ('mk2_cannon'), ('ion_cannon'), ('plasma_cannon'), ('proton_cannon')) AS d(defense_type)
WHERE NOT EXISTS (
    SELECT 1 FROM planet.player_defenses pd
    WHERE pd.planet_id = p.id AND pd.defense_type = d.defense_type
);

INSERT INTO planet.moon_buildings (moon_galaxy, moon_system, moon_position, type, level)
SELECT m.galaxy, m.system, m.position, btype, 0
FROM fleet.moons m
CROSS JOIN (VALUES ('moon_base'), ('robotics_factory'), ('shipyard'), ('pioneer_lab')) AS t(btype)
WHERE NOT EXISTS (
    SELECT 1 FROM planet.moon_buildings mb
    WHERE mb.moon_galaxy = m.galaxy
      AND mb.moon_system = m.system
      AND mb.moon_position = m.position
      AND mb.type = t.btype
);
