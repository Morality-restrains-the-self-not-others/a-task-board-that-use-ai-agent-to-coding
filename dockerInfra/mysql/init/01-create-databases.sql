-- MySQL init: create all service databases and application user
-- This runs on first container start via docker-entrypoint-initdb.d

CREATE USER IF NOT EXISTS 'taskapp'@'%' IDENTIFIED BY 'taskapp123';
GRANT ALL PRIVILEGES ON *.* TO 'taskapp'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;

CREATE DATABASE IF NOT EXISTS task_auth       CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS task_bill       CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS task_budget     CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS task_referral   CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS git_oauth       CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS ai_provider     CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS task_project    CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS task_task       CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS task_cloud      CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS task_ai_comment CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS task_tenant     CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS container       CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

GRANT ALL PRIVILEGES ON task_auth.*       TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON task_bill.*       TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON task_budget.*     TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON task_referral.*   TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON git_oauth.*       TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON ai_provider.*     TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON task_project.*    TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON task_task.*       TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON task_cloud.*      TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON task_ai_comment.* TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON task_tenant.*     TO 'taskapp'@'%';
GRANT ALL PRIVILEGES ON container.*       TO 'taskapp'@'%';
FLUSH PRIVILEGES;
