# Intent: 配置中的管理员邮箱在系统初始化时写入数据库

## 背景与目标

超级管理员登录邮箱的 SSOT 是 `conf/auth/task-auth/config.yaml` 的 `bootstrapAdmin.email`。`022_seed_bootstrap_admin.sql` 使用占位符 `__BOOTSTRAP_ADMIN_EMAIL__`，migrate 按 conf（含 conf-local）渲染后写入 `auth_login_method.identifier`。9999「初始化全部数据库」随后执行 `db/task-auth/init.sh` → `taskAuth bootstrap-admin` 幂等再同步同一邮箱。

## 范围与边界

- **范围内**：`conf/auth/task-auth/config.yaml` 的 `bootstrapAdmin.email`；`022_seed_bootstrap_admin.sql` 以 `__BOOTSTRAP_ADMIN_EMAIL__` 占位，`runDataMigrateFromDir` 与 `apply_datamigrate.sh` 按 conf（含 conf-local）渲染后入库；`taskAuth bootstrap-admin` 幂等再同步 identifier；`010_verify_super_admin.py` 读取同一配置做校验。
- **范围外**：前端 FAQ/`contactEmail`；管理员密码（仍为随机哈希 + 邮箱重置）；新 HTTP API；业务进程启动时自动迁移。

## 约束与风险

- 编辑落点：`conf/auth/task-auth/config.yaml`；本机覆盖 `conf-local/auth/task-auth/config.yaml`（ADR-0054 / 规则 42）。
- 运行时只读本服务目录（规则 29）；`confload.ReadAppConfig` 叠 conf-local。
- DDL/种子用户行仍在 `022_seed_bootstrap_admin.sql`；邮箱不以 SQL 字面量为 SSOT，migrate 时替换占位符。`bootstrap-admin` 对已应用的 022 仍幂等 UPDATE。
- 目标邮箱若已被其他活跃用户占用则失败，不得抢占。
- 空邮箱失败，禁止静默回退到硬编码。
- 应用进程不在启动路径写库；仅显式 `bootstrap-admin` CLI / 9999 init.sh。

## 验收标准

1. tracked `config.yaml` 含非空 `bootstrapAdmin.email`。
2. `022_seed_bootstrap_admin.sql` 的 INSERT identifier 为占位符，不含具体邮箱字面量。
3. 仅跑 `runDataMigrateFromDir` / `apply_datamigrate.sh` 后，`bootstrap-admin` 的 email identifier 等于 conf（含 conf-local）。
4. `taskAuth bootstrap-admin` 后 identifier 仍等于配置邮箱；重复执行幂等。
5. 配置邮箱已被其他用户占用时 CLI 非零退出。
6. `010_verify_super_admin.py` 校验的邮箱来自同一 conf 键，不再写死常量作为 SSOT。

## 实施计划

1. 意图与测试意图落盘。
2. conf 增加 `bootstrapAdmin.email`（默认保持现网 `author@example.com`）。
3. Go：从 conf 解析邮箱并 UPDATE `auth_login_method`；单测覆盖写入、幂等、空值、冲突。
4. Python 010 改读 conf；回归测断言不再以脚本常量当 SSOT。
5. 价值流 `dev-bootstrap-admin` 字段描述改为「conf 邮箱」。

## 角色与权限

无新 HTTP 端点。写库仅运维 CLI（9999 init / `run.sh bootstrap-admin`）。不改变 `super_admin` 授权模型。

## NFR 摘要

| 路径 | 分片键 | 可伸缩性 | 副作用 | 幂等 |
|------|--------|----------|--------|------|
| 9999 init → bootstrap-admin | 全局单例 `user_id=bootstrap-admin` | L0：平台仅一个 bootstrap 超管；升级触发=多环境独立库 | 写 `auth_login_method.identifier` | L2：键=`bootstrap-admin`+email 登录方式；重放 UPDATE 同一行 |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 系统初始化写入管理员邮箱 | — | — | `taskAuth` CLI `bootstrap-admin` | MySQL `auth_login_method` | 运维库初始化，无租户业务事实，无对应领域事件 |

## 变更记录

- 2026-09-01：新增。管理员邮箱从 SQL/脚本硬编码改为 conf，init 时写入。
- 2026-09-01：OPT-20260901-013 — 022 占位符由 migrate/apply_datamigrate 按 conf 渲染，去掉 SQL 与 conf 双写。
