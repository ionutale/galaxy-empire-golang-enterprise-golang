ALTER TABLE fleet.fleets
DROP COLUMN IF EXISTS cargo_metal,
DROP COLUMN IF EXISTS cargo_crystal,
DROP COLUMN IF EXISTS cargo_gas;
