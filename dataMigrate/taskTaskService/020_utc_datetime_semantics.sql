-- Naive DATETIME created_at stores UTC wall clock (no zone).
-- Comment INSERT writes UTC digits via formatMySQLUTCDateTime so loc=Local DSN
-- cannot convert to Asia/Shanghai. Do not bulk-convert historical local digits.

ALTER TABLE task_comments COMMENT='created_at stores UTC naive DATETIME';
