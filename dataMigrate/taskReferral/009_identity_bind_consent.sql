-- Applicant consent to bind account identifier and identity
-- for WeChat Pay consistency check (profit-sharing receiver).
-- Scale: one timestamp per application row; year increment
-- follows referral_code (much less than 1M). No partition.
-- Prefix referral_.
-- WeChat Pay: merchant must obtain user authorization before
-- transmitting identity and account identifier for checks.
-- https://pay.weixin.qq.com/doc/v3/merchant/4012528995

ALTER TABLE referral_code
ADD COLUMN identity_bind_consented_at DATETIME(6) NULL
COMMENT 'consent time for WeChat Pay identity bind';
