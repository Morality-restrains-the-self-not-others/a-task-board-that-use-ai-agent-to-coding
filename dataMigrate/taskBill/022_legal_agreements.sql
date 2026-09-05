-- 022_legal_agreements.sql
-- License Agreement + Privacy Policy tables (migrated from Django license_agreement + privacy_policy apps)
-- 2026-07-26: Django→Go Phase 2.5
-- 2026-08-03: Move rename-before-create to preserve old Django data (was conflict with 032).

-- Step 1: Rename old Django tables to billing_* prefix (preserve existing data).
-- Only renames if old table exists AND new billing_* table doesn't yet exist.
-- This runs BEFORE CREATE TABLE IF NOT EXISTS so old data is never stranded.
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='license_agreements') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_license_agreements'),'ALTER TABLE `license_agreements` RENAME TO `billing_license_agreements`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='user_license_agreement_consents') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_user_license_agreement_consents'),'ALTER TABLE `user_license_agreement_consents` RENAME TO `billing_user_license_agreement_consents`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='privacy_policies') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_privacy_policies'),'ALTER TABLE `privacy_policies` RENAME TO `billing_privacy_policies`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='user_privacy_policy_consents') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_user_privacy_policy_consents'),'ALTER TABLE `user_privacy_policy_consents` RENAME TO `billing_user_privacy_policy_consents`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;

-- Step 2: Create tables if they don't exist (greenfield deployment fallback).
CREATE TABLE IF NOT EXISTS billing_license_agreements (
    id          VARCHAR(64) PRIMARY KEY,
    title       TEXT NOT NULL DEFAULT (''),
    content     TEXT NOT NULL DEFAULT (''),
    version     TEXT NOT NULL DEFAULT (''),
    document_kind TEXT NOT NULL DEFAULT ('service'), -- service | recharge_points
    is_active   INTEGER NOT NULL DEFAULT 1,
    is_material_change INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_at  TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_la_document_kind ON billing_license_agreements(document_kind(255));
CREATE INDEX idx_la_active ON billing_license_agreements(is_active);

CREATE TABLE IF NOT EXISTS billing_user_license_agreement_consents (
    id                    VARCHAR(64) PRIMARY KEY,
    user_id               TEXT NOT NULL DEFAULT (''),
    license_agreement_id  VARCHAR(64) NOT NULL REFERENCES billing_license_agreements(id),
    consented_at          TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    context               TEXT NOT NULL DEFAULT (''), -- register_phone | register_email | login_password | login_phone_code | post_login_reconsent
    client_ip             TEXT NOT NULL DEFAULT (''),
    user_agent            TEXT NOT NULL DEFAULT ('')
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_ulac_user ON billing_user_license_agreement_consents(user_id(255));
CREATE INDEX idx_ulac_agreement ON billing_user_license_agreement_consents(license_agreement_id);
CREATE UNIQUE INDEX idx_ulac_unique ON billing_user_license_agreement_consents(user_id(255), license_agreement_id);

CREATE TABLE IF NOT EXISTS billing_privacy_policies (
    id          VARCHAR(64) PRIMARY KEY,
    title       TEXT NOT NULL DEFAULT (''),
    content     TEXT NOT NULL DEFAULT (''),
    version     TEXT NOT NULL DEFAULT (''),
    is_active   INTEGER NOT NULL DEFAULT 1,
    is_material_change INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_at  TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_pp_active ON billing_privacy_policies(is_active);

CREATE TABLE IF NOT EXISTS billing_user_privacy_policy_consents (
    id                VARCHAR(64) PRIMARY KEY,
    user_id           TEXT NOT NULL DEFAULT (''),
    privacy_policy_id VARCHAR(64) NOT NULL REFERENCES billing_privacy_policies(id),
    consented_at      TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    context           TEXT NOT NULL DEFAULT (''),
    client_ip         TEXT NOT NULL DEFAULT (''),
    user_agent        TEXT NOT NULL DEFAULT ('')
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_uppc_user ON billing_user_privacy_policy_consents(user_id(255));
CREATE INDEX idx_uppc_policy ON billing_user_privacy_policy_consents(privacy_policy_id);
CREATE UNIQUE INDEX idx_uppc_unique ON billing_user_privacy_policy_consents(user_id(255), privacy_policy_id);
