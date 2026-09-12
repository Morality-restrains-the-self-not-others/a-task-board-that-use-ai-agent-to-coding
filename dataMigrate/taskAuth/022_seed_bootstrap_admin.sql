-- 022: 种子默认超级管理员凭证
-- 用户行 / 超管标记在本文件 INSERT IGNORE。
-- 登录邮箱 identifier 为占位符，由 runDataMigrateFromDir / apply_datamigrate.sh
-- 按 conf bootstrapAdmin.email 渲染。ensureBootstrapAdminSeeded 再幂等同步。
--
-- ⚠️ 安全策略（OPT-20260824-001 + 初始化随机密码）：
--    - password_hash 使用哨兵 __BOOTSTRAP_ADMIN_PASSWORD_PENDING__；
--      taskAuth bootstrap-admin（9999 init.sh）首次执行时生成每环境独立的
--      crypto/rand 密码并写入 bcrypt（明文不落盘、不打印）。
--    - 管理员首次使用请走「忘记密码」→ 邮箱重置链接流程设置新密码
--      （taskAuth 已支持：send-password-reset-link → EMAIL_SENT Kafka/SMTP → 重置链接）。
--    - 生产/开发环境均不得改回已知明文密码；弱哈希由
--      dataMigrate/taskAuth/010_verify_super_admin.py 检测并自动轮换为新随机哈希。
--    - must_change_password=1 保留：密码重置成功后 taskAuth 自动清除该标记。
--
-- 幂等：使用 INSERT IGNORE，重复执行无害。
-- 既有环境（已存在旧已知密码哈希 / 历史共享哈希）由 bootstrap-admin 与 010 检测并轮换。

-- 超级管理员用户
INSERT IGNORE INTO auth_user (id, password, last_login, is_superuser, is_staff, is_active, is_tenant, date_joined, must_change_password)
VALUES ('bootstrap-admin', '', NULL, 1, 1, 1, 0, NOW(), 1);

-- Email 登录方式：password_hash 为待填哨兵，由 bootstrap-admin 替换为真随机 bcrypt。
-- identifier 为占位符：apply_datamigrate.sh 与 taskAuth runDataMigrateFromDir
-- 按 conf/auth/task-auth bootstrapAdmin.email（含 conf-local）替换后再执行。
-- 禁止在本文件写死邮箱或写死 bcrypt 哈希。
-- content_type_id=4 对应 auth_django_content_type 中的 accounts/user
INSERT IGNORE INTO auth_login_method (id, content_type_id, object_id, method_type, identifier, password_hash, is_verified, created_at, updated_at)
VALUES (1000000000000000001, 4, 'bootstrap-admin', 'email', '__BOOTSTRAP_ADMIN_EMAIL__', '__BOOTSTRAP_ADMIN_PASSWORD_PENDING__', 1, NOW(), NOW());

-- 超级管理员权限
INSERT IGNORE INTO auth_super_admin (user_id)
VALUES ('bootstrap-admin');
