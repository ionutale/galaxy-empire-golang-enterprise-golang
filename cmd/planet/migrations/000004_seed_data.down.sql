-- Only remove seeded moon building rows (level 0 defaults), not player-upgraded ones
DELETE FROM planet.moon_buildings
WHERE type IN ('moon_base', 'robotics_factory', 'shipyard', 'pioneer_lab')
  AND level = 0;

-- Only remove seeded defense rows with quantity 0 (untouched defaults)
DELETE FROM planet.player_defenses
WHERE defense_type IN ('rocket_launcher', 'light_laser', 'heavy_laser', 'mk2_cannon', 'ion_cannon', 'plasma_cannon', 'proton_cannon')
  AND quantity = 0;

-- Only remove seeded ship rows with quantity 0 (untouched defaults)
DELETE FROM planet.player_ships
WHERE ship_type IN ('cargo', 'large_cargo', 'recycler', 'espionage_probe', 'colony_ship', 'solar_satellite', 'light_fighter', 'heavy_fighter', 'cruiser', 'battleship', 'dreadnought', 'bomber')
  AND quantity = 0;

-- Only remove seeded technology rows at their seed level (level 3 energy_tech)
DELETE FROM planet.player_technologies
WHERE type = 'energy_tech' AND level = 3;

-- Only remove seeded building rows at their seed levels (level 0 or 1)
DELETE FROM planet.buildings
WHERE type IN ('robotics_factory', 'nanite_factory', 'terraformer', 'fusion_reactor', 'shipyard', 'missile_silo')
  AND level = 1;
DELETE FROM planet.buildings
WHERE type IN ('small_shield_dome', 'large_shield_dome')
  AND level = 0;

-- Only remove seeded player_progress rows with no accumulated data
DELETE FROM planet.player_progress
WHERE vip_points = 0 AND total_resources_produced = 0;
