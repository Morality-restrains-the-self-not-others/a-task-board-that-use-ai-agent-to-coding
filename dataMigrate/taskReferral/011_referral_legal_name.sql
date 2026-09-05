-- Personal legal name on referral applications (WeChat real-name for profit sharing).
-- Scale: one name per application row; follows referral_code growth. No partition.
-- Prefix referral_.

ALTER TABLE referral_code
  ADD COLUMN legal_name VARCHAR(64) NOT NULL DEFAULT ''
  COMMENT 'WeChat real-name used for profit sharing; mismatch causes sharing to fail';
