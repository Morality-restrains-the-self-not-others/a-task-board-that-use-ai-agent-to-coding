# Implementation Plan: 账号 init 双库对齐

> Design: `docs/superpowers/specs/2026-05-31-account-init-taskauth-architecture-design.md`

## Tasks

- [x] **1** `db/registry.yaml`: task-auth order 9，saas order 10（init 时 auth 先于 saas）
- [x] **2** `db/saas/migrate.sh`: export `TASKAUTH_USE_SEPARATE_DB=1`；末尾 drop+verify
- [x] **3** `taskAuth/src/bootstrap_admin.go` + `main.go` 子命令 `bootstrap-admin`
- [x] **4** `taskAuth/src/bootstrap_admin_test.go`: 迁移后 bootstrap 幂等
- [x] **5** `db/task-auth/init.sh`: 调用 bootstrap-admin
- [x] **6** `taskAuth/run.sh`: 增加 bootstrap-admin 命令
- [x] **7** 重构 `create_admin_ruandao.py`: 读 auth.db user id，仅写 saas SuperAdmin
- [x] **8** `db/saas/init.sh`: 末尾 drop+verify 兜底
- [x] **9** `value-stream.yaml`: 新增 dev-bootstrap-admin step
- [x] **10** 验证: `go test ./...` in taskAuth + init.sh smoke + verify_auth_tables_dropped
