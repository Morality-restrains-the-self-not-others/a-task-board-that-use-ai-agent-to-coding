-- Migration 003: Fix referral_code.id missing AUTO_INCREMENT.
-- Error 1364 (HY000): Field 'id' doesn't have a default value
-- The original CREATE TABLE had `id BIGINT NOT NULL PRIMARY KEY` without AUTO_INCREMENT,
-- but the Go INSERT statements do not provide an explicit id value.
-- Idempotent: MODIFY COLUMN with AUTO_INCREMENT is safe to re-run.

ALTER TABLE referral_code MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT;
