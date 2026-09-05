-- taskBill 计费表（从 Saas_project billing 抽取，独占 billing.db；tenant_id 不引用 accounts_company）
-- [MySQL compat] removed: PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS billing_pricing_package (
  id bigint NOT NULL PRIMARY KEY,
  package_number integer NOT NULL UNIQUE CHECK (package_number >= 0),
  valid_from date NOT NULL,
  valid_to date NULL,
  normal_task_points bigint NOT NULL,
  programming_task_points bigint NOT NULL,
  normal_task_renewal_points_per_month bigint NOT NULL DEFAULT 1,
  programming_task_renewal_points_per_month bigint NOT NULL DEFAULT 8,
  created_at datetime NOT NULL,
  updated_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX billing_pricing_package_valid_from ON billing_pricing_package (valid_from);

CREATE TABLE IF NOT EXISTS billing_unit (
  id bigint NOT NULL PRIMARY KEY,
  unit_type varchar(20) NOT NULL UNIQUE,
  name varchar(100) NOT NULL,
  price bigint NOT NULL,
  unit varchar(20) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  created_at datetime NOT NULL,
  updated_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS billing_account (
  id bigint NOT NULL PRIMARY KEY,
  tenant_id bigint NOT NULL UNIQUE,
  balance bigint NOT NULL DEFAULT 0,
  pricing_package_id bigint NOT NULL REFERENCES billing_pricing_package (id),
  locked_post_creation_points bigint NOT NULL DEFAULT 3,
  locked_server_start_points bigint NOT NULL DEFAULT 30,
  locked_normal_renewal_points_per_month bigint NOT NULL DEFAULT 1,
  locked_programming_renewal_points_per_month bigint NOT NULL DEFAULT 8,
  excluded_pricing_package_ids varchar(4096) NOT NULL DEFAULT '[]',
  created_at datetime NOT NULL,
  updated_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX billing_account_pricing_package_id ON billing_account (pricing_package_id);

CREATE TABLE IF NOT EXISTS billing_transaction (
  id bigint NOT NULL PRIMARY KEY,
  account_id bigint NOT NULL REFERENCES billing_account (id),
  transaction_type varchar(20) NOT NULL,
  amount bigint NOT NULL,
  balance_before bigint NOT NULL,
  balance_after bigint NOT NULL,
  points_source_type varchar(32) NULL,
  project_id varchar(36) NULL,
  user_id varchar(36) NULL,
  workspace_id varchar(36) NULL,
  task_id varchar(36) NULL,
  billing_unit_id bigint NULL REFERENCES billing_unit (id),
  usage_amount decimal NOT NULL DEFAULT 0,
  description varchar(255) NULL,
  transaction_id varchar(100) NOT NULL UNIQUE,
  created_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX billing_tra_account_created ON billing_transaction (account_id, created_at);
CREATE INDEX billing_tra_project_created ON billing_transaction (project_id, created_at);
CREATE INDEX billing_tra_user_created ON billing_transaction (user_id, created_at);
CREATE INDEX billing_tra_workspace_created ON billing_transaction (workspace_id, created_at);
CREATE INDEX billing_tra_task_created ON billing_transaction (task_id, created_at);

CREATE TABLE IF NOT EXISTS billing_usage (
  id bigint NOT NULL PRIMARY KEY,
  account_id bigint NOT NULL REFERENCES billing_account (id),
  billing_unit_id bigint NOT NULL REFERENCES billing_unit (id),
  amount decimal NOT NULL,
  project_id varchar(36) NULL,
  user_id varchar(36) NULL,
  workspace_id varchar(36) NULL,
  task_id varchar(36) NULL,
  description varchar(255) NULL,
  usage_time datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX billing_usa_account_time ON billing_usage (account_id, usage_time);
CREATE INDEX billing_usa_billing_time ON billing_usage (billing_unit_id, usage_time);
CREATE INDEX billing_usa_project_time ON billing_usage (project_id, usage_time);
CREATE INDEX billing_usa_user_time ON billing_usage (user_id, usage_time);
CREATE INDEX billing_usa_workspace_time ON billing_usage (workspace_id, usage_time);
CREATE INDEX billing_usa_task_time ON billing_usage (task_id, usage_time);

CREATE TABLE IF NOT EXISTS billing_idempotency_key (
  id bigint NOT NULL PRIMARY KEY,
  `key` varchar(200) NOT NULL UNIQUE,
  created_at datetime NOT NULL,
  transaction_id bigint NOT NULL UNIQUE REFERENCES billing_transaction (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS billing_outbox_message (
  id bigint NOT NULL PRIMARY KEY,
  billing_transaction_id bigint NOT NULL REFERENCES billing_transaction (id),
  event_type varchar(64) NOT NULL,
  payload text NOT NULL,
  status varchar(16) NOT NULL DEFAULT 'pending',
  created_at datetime NOT NULL,
  sent_at datetime NULL,
  last_error text NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX billing_out_status_created ON billing_outbox_message (status, created_at);

CREATE TABLE IF NOT EXISTS taskbill_schema_migrations (
  name VARCHAR(255) PRIMARY KEY,
  applied_at VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
