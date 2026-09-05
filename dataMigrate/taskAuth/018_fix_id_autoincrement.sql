-- Migration 018: Add AUTO_INCREMENT to id columns missing it.
-- Preventive fix — Go code currently provides explicit snowflake IDs for these tables,
-- but the schema should have AUTO_INCREMENT as a safety net for any future INSERTs
-- that might omit the id column.
-- Related: taskReferral/003 — same fix for referral_code (active bug there, Error 1364).
-- Idempotent: MODIFY COLUMN with AUTO_INCREMENT is safe to re-run.

ALTER TABLE auth_kyc_audit_log            MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT;
ALTER TABLE auth_aml_screening_record      MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT;
ALTER TABLE auth_sms_verification_code     MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT;
ALTER TABLE auth_email_invite_delivery_attempt MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT;
