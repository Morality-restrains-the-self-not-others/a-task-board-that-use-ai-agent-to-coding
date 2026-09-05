-- Store referrer legal name for WeChat profit-sharing receiver add / share requests.
-- Scale: one row per referrer. No partition.

ALTER TABLE billing_profit_sharing_receiver
  ADD COLUMN legal_name VARCHAR(64) NOT NULL DEFAULT ''
  COMMENT 'WeChat real-name; empty means omit name on WeChat APIs';
