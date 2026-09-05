# 测试意图：配置中的管理员邮箱在系统初始化时写入数据库

## 覆盖的功能意图

`docs/intents/platform/bootstrap_admin_email_from_conf.intent.md`

## 测试目标

确认 `bootstrapAdmin.email` 是 SSOT，系统初始化 CLI 把它写入 `auth_login_method`，且校验脚本读同一配置。

## 测试分层

- 单元：Go `resolveBootstrapAdminEmail` / `applyBootstrapAdminEmail`；Python `load_bootstrap_admin_email`
- 集成：`ensureBootstrapAdminSeeded` 对测试 MySQL 幂等写入

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 从 `conf/auth/task-auth/config.yaml`（含 conf-local）解析邮箱 | 非空且含 `@` |
| T2 | 空邮箱调用 apply | 返回 error，不写库 |
| T3 | seed 后 apply 配置外的测试邮箱 | `bootstrap-admin` 的 email identifier 等于该邮箱 |
| T4 | 再次 `ensureBootstrapAdminSeeded` | 仍 1 条 email login_method；user_id 不变 |
| T5 | 目标邮箱已被其他用户占用 | apply 失败 |
| T6 | 010 脚本加载的管理员邮箱 | 等于 conf 解析结果，而非仅脚本内字面量 SSOT |
| T7 | 022 SQL 文本 | identifier 为 `__BOOTSTRAP_ADMIN_EMAIL__`，VALUES 无具体邮箱 |
| T8 | `render_sql` 替换占位符 | 输出含给定邮箱且无占位符；含引号的邮箱失败 |
| T9 | 仅 `runDataMigrateFromDir`（不跑 applyBootstrapAdminEmail） | DB 中 identifier 等于 conf |

## 数据与环境

- Go：`dbload.OpenTestMySQL("task-auth")`；无 MySQL 则 Skip
- Python：读仓库 `conf/auth/task-auth/config.yaml`

## 通过标准

上表对应用例全部通过；`gofmt` / `py_compile` / YAML `safe_load` 通过。

## 对应测试文件

- `taskAuth/src/bootstrap_admin_test.go`
- `dataMigrate/taskAuth/test_010_verify_super_admin.py`
- `dataMigrate/taskAuth/test_bootstrap_admin_email.py`
