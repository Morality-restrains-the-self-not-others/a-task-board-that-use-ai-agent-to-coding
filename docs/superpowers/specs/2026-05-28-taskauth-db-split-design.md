# taskAuth Increment 6 — 认证表搬迁至 taskAuth/data/auth.db

## 目标

- Go taskAuth 独占 `taskAuth/data/auth.db`（`accounts_user`、`accounts_login_method`、`accounts_customtoken`、`django_content_type`）
- Django default 仍保留 `accounts_user`（业务 FK）；Go 创建用户后通过 `POST /api/internal/taskauth/sync-user/` 同步
- `CustomToken` / `LoginMethod` 经 `DATABASE_ROUTERS` 读写 taskauth 库（`TASKAUTH_USE_SEPARATE_DB=true`）

## 首次迁移

```bash
taskAuth/scripts/migrate_from_shared_db.sh
```

## 配置

`task2app/conf/port_config.json` → `taskAuth.databasePath`: `taskAuth/data/auth.db`

环境变量：`TASKAUTH_DATABASE_PATH`、`TASKAUTH_USE_SEPARATE_DB`
