-- skipped_unsubscribed is 20 chars; widen off the VARCHAR(20) knife-edge.
-- Idempotent MODIFY.
ALTER TABLE auth_email_registration_invite
  MODIFY COLUMN delivery_status VARCHAR(32) NOT NULL DEFAULT 'pending';
ALTER TABLE auth_email_invite_delivery_attempt
  MODIFY COLUMN delivery_status VARCHAR(32) NOT NULL;
