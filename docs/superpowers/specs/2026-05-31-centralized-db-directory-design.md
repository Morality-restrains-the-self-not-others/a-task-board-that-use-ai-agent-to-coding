# 集中式 SQLite 目录 `db/` — 设计文档

**日期:** 2026-05-31  
**状态:** 已批准  
**范围:** monorepo 根 `db/`、`db/registry.yaml`、Python/Go loader、6 个 SQLite 服务路径迁移

---

## 1. 背景与目标

### 1.1 问题

SQLite 文件分散在 6 处，配置入口不统一（`paths.conf`、`port_config.json`、各 `settings.py` 硬编码），备份与排障成本高。

### 1.2 目标

1. 全部 SQLite 归口到 monorepo 根 `db/<service>/`。
2. `db/registry.yaml` 为唯一路径真源。
3. 统一 `.sqlite3` 后缀。
4. 提供一次性迁移脚本 `db/migrate-existing.sh`。
5. 废弃 `paths.conf` 的 `DB_SQLITE_PATH` 与 `port_config.json` 的 `databasePath`。

### 1.3 非目标

- Redis/Kafka Docker volume
- 生产 PostgreSQL
- 表结构或业务逻辑变更

---

## 2. 目录布局

```
db/
├── README.md
├── registry.yaml
├── migrate-existing.sh
├── .gitignore
├── load/
│   ├── loader.py
│   ├── registry.go
│   └── go.mod
├── saas/saas.sqlite3
├── task-auth/auth.sqlite3
├── task-bill/billing.sqlite3
├── git-oauth/git-oauth.sqlite3
├── email/email.sqlite3
└── ai-provider/ai-provider.sqlite3
```

---

## 3. `registry.yaml`

```yaml
version: "1"

databases:
  saas:
    path: db/saas/saas.sqlite3
    owner: saas-backend
  task-auth:
    path: db/task-auth/auth.sqlite3
    owner: task-auth
  task-bill:
    path: db/task-bill/billing.sqlite3
    owner: task-bill
  git-oauth:
    path: db/git-oauth/git-oauth.sqlite3
    owner: git-oauth
  email:
    path: db/email/email.sqlite3
    owner: email
  ai-provider:
    path: db/ai-provider/ai-provider.sqlite3
    owner: ai-provider
```

**解析规则：**

- Monorepo 根：向上查找 `db/registry.yaml`（与 Go 服务 `task2app/conf/port_config.json` marker 一致）。
- 路径相对 monorepo 根。
- 环境变量覆盖：

| Key | 环境变量 |
|-----|----------|
| `saas` | `SAAS_DATABASE_PATH` |
| `task-auth` | `TASKAUTH_DATABASE_PATH` |
| `task-bill` | `TASKBILL_DATABASE_PATH` |
| `git-oauth` | `GITOAUTH_DATABASE_PATH` |
| `email` | `EMAIL_DATABASE_PATH` |
| `ai-provider` | `AI_PROVIDER_DATABASE_PATH` |

- Loader 在 dev 下 `mkdir -p` 父目录。

---

## 4. 服务接入

| 服务 | 旧路径 | registry key |
|------|--------|--------------|
| Django 主站 | `Saas_project/db.sqlite3` | `saas` |
| taskAuth | `taskAuth/data/auth.db` | `task-auth` |
| taskBill | `taskBill/data/billing.db` | `task-bill` |
| gitOauth | `gitOauth/db.sqlite3` | `git-oauth` |
| Saas_email | `Saas_email/db.sqlite3` | `email` |
| Saas_Ai_Provider | `Saas_Ai_Provider/db.sqlite3` | `ai-provider` |

---

## 5. 迁移

`db/migrate-existing.sh` 幂等 `mv` 旧库至新路径（含 `-wal`/`-shm`）。

---

## 6. 价值流影响

无新业务流；`user-auth` / billing 字段语义不变，仅物理路径配置化。

---

## 7. 验收

1. `db/migrate-existing.sh` 在含旧库环境跑通。
2. runAll 全栈启动，health database 通过。
3. loader 单元测试 + env 覆盖测试通过。
4. `rg` 无残留旧默认路径（除 migrate 脚本注释）。
