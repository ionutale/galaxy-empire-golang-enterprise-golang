CREATE SCHEMA IF NOT EXISTS galaxy;

CREATE TABLE IF NOT EXISTS galaxy.galaxies (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO galaxy.galaxies (id, name)
SELECT g.id, 'Galaxy ' || g.id::text
FROM generate_series(1, 9) AS g(id)
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS galaxy.systems (
    id         SERIAL PRIMARY KEY,
    galaxy_id  INT NOT NULL REFERENCES galaxy.galaxies(id),
    system_num INT NOT NULL CHECK (system_num BETWEEN 1 AND 499),
    UNIQUE(galaxy_id, system_num)
);

INSERT INTO galaxy.systems (galaxy_id, system_num)
SELECT g.id, s.num
FROM galaxy.galaxies g
CROSS JOIN generate_series(1, 499) AS s(num)
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS galaxy.positions (
    id          SERIAL PRIMARY KEY,
    system_id   INT NOT NULL REFERENCES galaxy.systems(id),
    position_num INT NOT NULL CHECK (position_num BETWEEN 1 AND 15),
    UNIQUE(system_id, position_num)
);

INSERT INTO galaxy.positions (system_id, position_num)
SELECT s.id, p.num
FROM galaxy.systems s
CROSS JOIN generate_series(1, 15) AS p(num)
ON CONFLICT DO NOTHING;
