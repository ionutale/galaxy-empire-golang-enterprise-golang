-- Fix: only drop combat-specific tables, never touch fleet schema
DROP TABLE IF EXISTS combat.moon_coordinates;
DROP TABLE IF EXISTS combat.combat_reports;
DROP SCHEMA IF EXISTS combat CASCADE;
