# 测试意图：配置中的管理员邮箱在系统初始化时写入数据库

## 覆盖的功能意图

`docs/intents/platform/bootstrap_admin_email_from_conf.intent.md`

## 测试目标

确认 `bootstrapAdmin.email` 是 SSOT，系统初始化须先设置该邮箱，再生成随机密码写入 `auth_login_method`。

## 测试分层

- 单元：Go `resolveBootstrapAdminEmail` / `applyBootstrapAdminEmail` / `ensureBootstrapAdminRandomPassword` / `needsBootstrapAdminPasswordRotation`；Python `load_bootstrap_admin_email` / `_needs_password_rotation`；runAll `writeBootstrapAdminEmailConfLocal`
- 集成：`ensureBootstrapAdminSeeded` 对测试 MySQL 幂等写入邮箱 + 随机密码；9999 API round-trip

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 从 `conf/auth/task-auth/config.yaml`（含 conf-local）解析邮箱 | 非空且含 `@` |
| T2 | 空邮箱调用 apply | 返回 error，不写库 |
| T3 | seed 后 apply 配置外的测试邮箱 | `bootstrap-admin` 的 email identifier 等于该邮箱 |
| T4 | 再次 `ensureBootstrapAdminSeeded` | 仍 1 条 email login_method；user_id 不变；password_hash 不变 |
| T5 | 目标邮箱已被其他用户占用 | apply 失败 |
| T6 | 010 脚本加载的管理员邮箱 | 等于 conf 解析结果，而非仅脚本内字面量 SSOT |
| T7 | 022 SQL 文本 | identifier 为 `__BOOTSTRAP_ADMIN_EMAIL__`；password_hash 为 `__BOOTSTRAP_ADMIN_PASSWORD_PENDING__`；无具体邮箱、无共享 bcrypt |
| T8 | `render_sql` 替换占位符 | 输出含给定邮箱且无占位符；含引号的邮箱失败 |
| T9 | 仅 `runDataMigrateFromDir`（不跑 applyBootstrapAdminEmail） | DB 中 identifier 等于 conf；password_hash 仍为哨兵直至 bootstrap-admin |
| T10 | bootstrap-admin 后密码 | 合法 bcrypt；非哨兵/非弱/非历史共享；`must_change_password=1` |
| T11 | 9999 POST `/api/dev/bootstrap-admin-email` | 写入 conf-local 并可 GET 读回；空邮箱 400 |
| T12 | Status UI | 含「请先设置超级管理员邮箱」与 `/api/dev/bootstrap-admin-email` |

## 数据与环境

- Go：`dbload.OpenTestMySQL("task-auth")`；无 MySQL 则 Skip
- Python：读仓库 `conf/auth/task-auth/config.yaml`
- runAll：`MONOREPO_ROOT` tempdir + `RUNALL_ALLOW_DEV_DB_RESET=1`

## 通过标准

上表对应用例全部通过；`gofmt` / `py_compile` / YAML `safe_load` 通过。

## 对应测试文件

- `taskAuth/src/bootstrap_admin_test.go`
- `taskAuth/src/auth_password_reset_must_change_test.go`
- `dataMigrate/taskAuth/test_010_verify_super_admin.py`
- `dataMigrate/taskAuth/test_bootstrap_admin_email.py`
- `runAll/src/ui_bootstrap_admin_email_test.go`
- `runAll/src/ui_test.go`（嵌入 UI 字符串护栏）
