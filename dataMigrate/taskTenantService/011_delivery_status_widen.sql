-- skipped_unsubscribed (20 chars) exceeds VARCHAR(16).
-- Idempotent MODIFY: re-run keeps VARCHAR(32).
ALTER TABLE tenant_invitation
  MODIFY COLUMN delivery_status VARCHAR(32) NOT NULL DEFAULT 'none';
