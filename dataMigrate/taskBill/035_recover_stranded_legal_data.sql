-- 035_recover_stranded_legal_data.sql
-- Recovery: If 022_legal_agreements.sql ran before the 2026-08-03 fix (create-then-rename, wrong order),
-- the old Django tables (license_agreements, privacy_policies, etc.) may still contain data that was
-- never migrated into the billing_* tables. The rename in 032 was skipped because billing_* already
-- existed from 022.
--
-- This migration copies any stranded data from old Django tables into billing_* tables.
-- Idempotent: uses INSERT IGNORE to skip rows that already exist in the target.
-- Uses dynamic SQL (PREPARE/EXECUTE) so MySQL doesn't parse table references when old tables are absent.
--
-- If old tables don't exist or new tables already have data, this is a harmless no-op.

-- ── License Agreements ─────────────────────────────────────────────────────

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='license_agreements')
  AND EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_license_agreements'),
  'INSERT IGNORE INTO billing_license_agreements (id, title, content, version, document_kind, is_active, is_material_change, created_at, updated_at) SELECT id, title, content, version, COALESCE(document_kind, ''service''), COALESCE(is_active, 1), COALESCE(is_material_change, 0), COALESCE(created_at, CURRENT_TIMESTAMP), COALESCE(updated_at, CURRENT_TIMESTAMP) FROM license_agreements',
  'SELECT 1'
);PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='user_license_agreement_consents')
  AND EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_user_license_agreement_consents'),
  'INSERT IGNORE INTO billing_user_license_agreement_consents (id, user_id, license_agreement_id, consented_at, context, client_ip, user_agent) SELECT id, user_id, license_agreement_id, COALESCE(consented_at, CURRENT_TIMESTAMP), COALESCE(context, ''''), COALESCE(client_ip, ''''), COALESCE(user_agent, '''') FROM user_license_agreement_consents',
  'SELECT 1'
);PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;

-- ── Privacy Policies ───────────────────────────────────────────────────────

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='privacy_policies')
  AND EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_privacy_policies'),
  'INSERT IGNORE INTO billing_privacy_policies (id, title, content, version, is_active, is_material_change, created_at, updated_at) SELECT id, title, content, version, COALESCE(is_active, 1), COALESCE(is_material_change, 0), COALESCE(created_at, CURRENT_TIMESTAMP), COALESCE(updated_at, CURRENT_TIMESTAMP) FROM privacy_policies',
  'SELECT 1'
);PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='user_privacy_policy_consents')
  AND EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_user_privacy_policy_consents'),
  'INSERT IGNORE INTO billing_user_privacy_policy_consents (id, user_id, privacy_policy_id, consented_at, context, client_ip, user_agent) SELECT id, user_id, privacy_policy_id, COALESCE(consented_at, CURRENT_TIMESTAMP), COALESCE(context, ''''), COALESCE(client_ip, ''''), COALESCE(user_agent, '''') FROM user_privacy_policy_consents',
  'SELECT 1'
);PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
