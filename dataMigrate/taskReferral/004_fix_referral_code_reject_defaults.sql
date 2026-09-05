-- Migration 004: Fix reject_reason / reviewed_by missing DEFAULT.
-- Error 1364 (HY000): Field 'reject_reason' doesn't have a default value
-- applyReferralCode INSERT omits these until rejected/reviewed; pending rows
-- must still satisfy NOT NULL.
-- TEXT cannot carry DEFAULT on this MySQL/MariaDB build → use VARCHAR(512).
-- Idempotent: MODIFY with DEFAULT '' is safe to re-run.

ALTER TABLE referral_code
  MODIFY COLUMN reject_reason VARCHAR(512) NOT NULL DEFAULT '',
  MODIFY COLUMN reviewed_by VARCHAR(512) NOT NULL DEFAULT '';
