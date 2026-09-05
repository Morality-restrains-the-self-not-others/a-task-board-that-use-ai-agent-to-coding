-- 022: 种子默认超级管理员凭证
-- 用户行 / 超管标记在本文件 INSERT IGNORE。
-- 登录邮箱 identifier 为占位符，由 runDataMigrateFromDir / apply_datamigrate.sh
-- 按 conf bootstrapAdmin.email 渲染。ensureBootstrapAdminSeeded 再幂等同步。
--
-- ⚠️ 安全策略（OPT-20260824-001）：管理员密码为随机生成，明文已销毁。
--    - 下方 password_hash 对应一个 256-bit 熵的随机密码（43 字符 base64），
--      生成后明文未保存、未打印、未入库，任何人（含开发/部署）均不知晓。
--    - 管理员首次使用请走「忘记密码」→ 邮箱重置链接流程设置新密码
--      （taskAuth 已支持：send-password-reset-link → EMAIL_SENT Kafka/SMTP → 重置链接）。
--    - 生产/开发环境均不得改回已知明文密码；如需重新轮换，参考
--      dataMigrate/taskAuth/010_verify_super_admin.py 顶部的轮换说明。
--    - must_change_password=1 保留：密码重置成功后 taskAuth 自动清除该标记。
--
-- 幂等：使用 INSERT IGNORE，重复执行无害。
-- 既有环境（已存在旧已知密码哈希）由 010_verify_super_admin.py 检测并自动轮换。

-- 超级管理员用户
INSERT IGNORE INTO auth_user (id, password, last_login, is_superuser, is_staff, is_active, is_tenant, date_joined, must_change_password)
VALUES ('bootstrap-admin', '', NULL, 1, 1, 1, 0, NOW(), 1);

-- Email 登录方式（bcrypt hash 对应随机密码，cost=12，明文未知）
-- 与 010_verify_super_admin.py 中 RANDOM_ADMIN_PASSWORD_HASH 必须保持一致。
-- identifier 为占位符：apply_datamigrate.sh 与 taskAuth runDataMigrateFromDir
-- 按 conf/auth/task-auth bootstrapAdmin.email（含 conf-local）替换后再执行。
-- 禁止在本文件写死邮箱。
-- content_type_id=4 对应 auth_django_content_type 中的 accounts/user
INSERT IGNORE INTO auth_login_method (id, content_type_id, object_id, method_type, identifier, password_hash, is_verified, created_at, updated_at)
VALUES (1000000000000000001, 4, 'bootstrap-admin', 'email', '__BOOTSTRAP_ADMIN_EMAIL__', '$2a$12$reqIRh/zFv.aez0dNtwWJ.kED6CpqFsppSm8pBOk6Vd9n8hKOK2A2', 1, NOW(), NOW());

-- 超级管理员权限
INSERT IGNORE INTO auth_super_admin (user_id)
VALUES ('bootstrap-admin');
