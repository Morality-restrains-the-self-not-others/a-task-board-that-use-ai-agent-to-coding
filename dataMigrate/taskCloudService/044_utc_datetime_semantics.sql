-- Naive DATETIME created_at/updated_at store UTC wall clock (no zone).
-- Application writes UTC digits (formatMySQLUTCDateTime / UTC_TIMESTAMP).
-- Do not bulk-convert historical rows that may still be loc=Local digits.

ALTER TABLE cloud_server_configs COMMENT='created_at/updated_at store UTC naive DATETIME';
ALTER TABLE cloud_server_events COMMENT='created_at/updated_at store UTC naive DATETIME';
ALTER TABLE cloud_comment_container_bindings COMMENT='created_at/updated_at store UTC naive DATETIME';
