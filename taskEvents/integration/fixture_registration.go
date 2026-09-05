//go:build integration

package integration

// RegistrationChainSchema is minimal SQLite for USER_CREATED → COMPANY_CREATED → WORKSPACE_CREATED.
func RegistrationChainSchema(userID string, username string) string {
	return `
CREATE TABLE tenant_company (
  id bigint PRIMARY KEY, name varchar(255) NOT NULL, creator_id varchar(36),
  created_at datetime NOT NULL, updated_at datetime NOT NULL
);
CREATE TABLE tenant_company_member (
  id bigint PRIMARY KEY, user_id varchar(36) NOT NULL, company_id bigint NOT NULL,
  is_admin bool NOT NULL, is_active bool NOT NULL,
  created_at datetime NOT NULL, updated_at datetime NOT NULL
);

CREATE TABLE auth_django_content_type (id integer PRIMARY KEY, app_label varchar(100), model varchar(100));
INSERT INTO auth_django_content_type (id, app_label, model) VALUES (18, 'projects', 'deliverablesystem'), (62, 'column_systems', 'progresssystem');

CREATE TABLE projects_deliverablesystem (id bigint PRIMARY KEY, name varchar(255) UNIQUE, description text, is_default bool, is_system bool, created_at datetime, updated_at datetime);
INSERT INTO projects_deliverablesystem (id, name, description, is_default, is_system, created_at, updated_at)
  VALUES (849094340575526912, '全局默认交付物体系', '', 1, 1, '2026-01-01', '2026-01-01');

CREATE TABLE column_systems_progresssystem (id bigint PRIMARY KEY, name varchar(100), is_default bool, created_at datetime, updated_at datetime);
INSERT INTO column_systems_progresssystem (id, name, is_default, created_at, updated_at)
  VALUES (1000000000000000100, '系统默认进度体系', 1, '2026-01-01', '2026-01-01');

CREATE TABLE projects_deliverablesystem_default_tenant (
  id bigint PRIMARY KEY, tenant_id varchar(36) UNIQUE, object_id varchar(36), content_type_name varchar(100),
  created_at datetime, updated_at datetime, content_type_id integer
);
CREATE TABLE column_systems_progresscolumn_default_tenant (
  id bigint PRIMARY KEY, tenant_id varchar(36) UNIQUE, object_id varchar(36), content_type_name varchar(100),
  created_at datetime, updated_at datetime, content_type_id integer
);
CREATE TABLE projects_workspace (
  id bigint PRIMARY KEY, name varchar(255), description text, deliverable_system_id varchar(255),
  deliverable_system_from varchar(20), is_default bool, created_at datetime, updated_at datetime,
  company_id bigint, task_archive_tier varchar(16)
);
CREATE TABLE column_systems_progresssystem_workspace (
  id bigint PRIMARY KEY, tenant_id varchar(36), workspace_id varchar(36), object_id varchar(36),
  content_type_name varchar(100), created_at datetime, updated_at datetime, content_type_id integer
);
CREATE TABLE projects_deliverablesystem_workspace (
  id bigint PRIMARY KEY, tenant_id varchar(36), workspace_id varchar(36), object_id varchar(36),
  content_type_name varchar(100), created_at datetime, updated_at datetime, content_type_id integer
);
CREATE TABLE projects_workspaceaccess (
  id bigint PRIMARY KEY, role varchar(20), created_at datetime, updated_at datetime, user_id varchar(36), workspace_id bigint
);
CREATE TABLE auth_login_method (
  id bigint PRIMARY KEY, content_type_id integer, object_id varchar(36), method_type varchar(20), identifier varchar(255)
);
INSERT INTO auth_login_method (id, content_type_id, object_id, method_type, identifier)
  VALUES (1, 4, '` + userID + `', 'username', '` + username + `');
`
}
