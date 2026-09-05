-- taskAuth 认证表（从 Saas_project 抽取，taskAuth 独占库）
-- [MySQL compat] removed: PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS auth_django_content_type (
  id integer NOT NULL PRIMARY KEY AUTO_INCREMENT,
  app_label varchar(100) NOT NULL,
  model varchar(100) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE UNIQUE INDEX django_content_type_app_label_model_76bd3d3b_uniq
  ON auth_django_content_type (app_label, model);

CREATE TABLE IF NOT EXISTS auth_user (
  id varchar(36) NOT NULL PRIMARY KEY,
  password varchar(128) NOT NULL,
  last_login datetime NULL,
  is_superuser TINYINT(1) NOT NULL,
  is_staff TINYINT(1) NOT NULL,
  is_active TINYINT(1) NOT NULL,
  date_joined datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_login_method (
  id bigint NOT NULL PRIMARY KEY,
  object_id varchar(36) NOT NULL,
  method_type varchar(20) NOT NULL,
  password_hash varchar(128) NOT NULL,
  is_verified TINYINT(1) NOT NULL,
  activation_token varchar(64) NULL UNIQUE,
  activation_token_expires_at datetime NULL,
  password_reset_token varchar(64) NULL UNIQUE,
  password_reset_token_expires_at datetime NULL,
  created_at datetime NOT NULL,
  updated_at datetime NOT NULL,
  content_type_id integer NOT NULL REFERENCES auth_django_content_type (id),
  binding_voided_at datetime NULL,
  phone_country_calling_code varchar(8) NULL,
  sms_verification_code_id bigint NULL,
  identifier varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX auth_login_method_object_id_ca72bdc2 ON auth_login_method (object_id);
CREATE INDEX auth_login_method_content_type_id_75db2029 ON auth_login_method (content_type_id);

CREATE TABLE IF NOT EXISTS auth_customtoken (
  `key` varchar(40) NOT NULL PRIMARY KEY,
  object_id varchar(36) NOT NULL,
  created datetime NOT NULL,
  content_type_id integer NOT NULL REFERENCES auth_django_content_type (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX auth_customtoken_content_type_id_0d03e798 ON auth_customtoken (content_type_id);

-- 种子：accounts.user content type（与 Django 默认 id=4 对齐；若冲突由迁移脚本修正）
INSERT IGNORE INTO auth_django_content_type (id, app_label, model) VALUES (4, 'accounts', 'user');

CREATE TABLE IF NOT EXISTS taskauth_schema_migrations (
  name VARCHAR(255) PRIMARY KEY,
  applied_at VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
