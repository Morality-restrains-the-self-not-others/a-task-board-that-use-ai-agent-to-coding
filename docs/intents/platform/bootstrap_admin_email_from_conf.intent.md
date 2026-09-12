# Intent: 配置中的管理员邮箱在系统初始化时写入数据库

## 背景与目标

超级管理员登录邮箱的 SSOT 是 `conf/auth/task-auth/config.yaml` 的 `bootstrapAdmin.email`（叠 `conf-local`）。`022_seed_bootstrap_admin.sql` 使用占位符 `__BOOTSTRAP_ADMIN_EMAIL__`，migrate 按 conf 渲染后写入 `auth_login_method.identifier`。

**初始化强制顺序**：
1. **先设置管理员邮箱** — 9999「初始化全部数据库」弹窗必填，写入 `conf-local/auth/task-auth/config.yaml`；亦可事先编辑 conf-local。
2. **再生成随机密码** — `db/task-auth/init.sh` → `taskAuth bootstrap-admin` 在同步邮箱后，用 `crypto/rand` 生成每环境独立密码并写入 bcrypt（明文不落盘、不打印）。022 种子仅含哨兵 `__BOOTSTRAP_ADMIN_PASSWORD_PENDING__`。

管理员首次登录须走「忘记密码」邮箱重置（`must_change_password=1`）。

## 范围与边界

- **范围内**：`bootstrapAdmin.email`；022 邮箱/密码哨兵；migrate 渲染邮箱；9999 `GET/POST /api/dev/bootstrap-admin-email`；`taskAuth bootstrap-admin` 同步邮箱 + 随机密码；`010_verify_super_admin.py` 校验/轮换弱哈希与哨兵。
- **范围外**：业务 HTTP 登录 API；业务进程启动时自动迁移。

## 约束与风险

- 编辑落点：tracked `conf/auth/task-auth/config.yaml`（开发占位）；运维真实邮箱优先 `conf-local/`（ADR-0054 / 规则 42/58）。
- 运行时只读本服务目录（规则 29）；`confload.ReadAppConfig` 叠 conf-local。
- 空邮箱失败，禁止静默跳过 9999 邮箱弹窗。
- 密码幂等：已是非弱、非哨兵、非历史共享哈希时不覆盖（避免抹掉运维已设密码）。
- 应用进程不在启动路径写库；仅显式 `bootstrap-admin` CLI / 9999 init.sh。

## 验收标准

1. tracked `config.yaml` 含 `bootstrapAdmin.email` 键；空/非法值使 resolve/apply 失败。
2. `022_seed_bootstrap_admin.sql` 的 identifier 为 `__BOOTSTRAP_ADMIN_EMAIL__`，password_hash 为 `__BOOTSTRAP_ADMIN_PASSWORD_PENDING__`，无具体邮箱字面量、无共享 bcrypt。
3. 9999 初始化前须经弹窗设置邮箱并 `POST /api/dev/bootstrap-admin-email` 写入 conf-local。
4. `taskAuth bootstrap-admin` 后 identifier 等于配置邮箱；password_hash 为合法 bcrypt 且不等于哨兵/弱明文/历史共享哈希；重复执行幂等（哈希不变）。
5. 配置邮箱已被其他用户占用时 CLI 非零退出。
6. `010_verify_super_admin.py` 校验的邮箱来自同一 conf 键；弱/哨兵/共享哈希触发新随机轮换。

## 角色与权限

无新业务 HTTP 端点。9999 开发 API 仅私网/本机。写库仅运维 CLI（9999 init / `run.sh bootstrap-admin`）。不改变 `super_admin` 授权模型。

## NFR 摘要

| 路径 | 分片键 | 可伸缩性 | 副作用 | 幂等 |
|------|--------|----------|--------|------|
| 9999 set email → conf-local | 全局单例文件 | L0 | 写 conf-local YAML | L2：同邮箱重写 |
| 9999 init → bootstrap-admin | 全局单例 `user_id=bootstrap-admin` | L0 | 写 identifier + password_hash | L2：邮箱 UPDATE 同值；密码仅哨兵/弱/共享时轮换 |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 系统初始化写入管理员邮箱与随机密码 | — | — | `taskAuth` CLI `bootstrap-admin` | MySQL `auth_login_method` | 运维库初始化，无租户业务事实，无对应领域事件 |

## 变更记录

- 2026-09-01：新增。管理员邮箱从 SQL/脚本硬编码改为 conf，init 时写入。
- 2026-09-01：OPT-20260901-013 — 022 占位符由 migrate/apply_datamigrate 按 conf 渲染，去掉 SQL 与 conf 双写。
- 2026-09-11：初始化强制先设邮箱（9999 弹窗 + conf-local）；密码改为每环境 bootstrap-admin 随机生成（022 仅哨兵）。
