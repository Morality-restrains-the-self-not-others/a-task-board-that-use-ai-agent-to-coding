# 实施计划: 开发环境一键重置全部数据库

> 设计: `docs/superpowers/specs/2026-05-31-dev-database-reset-design.md`  
> 价值流: `docs/superpowers/plans/2026-05-31-dev-database-reset-value-stream.md`  
> NFR: `docs/superpowers/plans/2026-05-31-dev-database-reset-nfr-clarification.md`  
> 领域: `runAll/src/domain/database_platform_reset.go`

## 任务清单

### Increment 1 — Registry + domain + saas 脚本

- [ ] **Task 1.1** 扩展 `db/registry.yaml`（`order`, `migrate_script`, `init_script`）  
  - 验证: `cd db/load && go test ./...`

- [ ] **Task 1.2** `db/load/registry_entries.go` + `LoadDatabaseEntries` 单测  
  - 验证: `go test ./...` in `db/load`

- [ ] **Task 1.3** `runAll/src/domain/database_platform_reset.go` + `_test.go`（mock 执行器）  
  - 验证: `cd runAll && go test ./src/domain/... -run DatabasePlatform`

- [ ] **Task 1.4** `db/saas/migrate.sh`, `db/saas/init.sh`  
  - 验证: 手工 `bash db/saas/migrate.sh`（空库）

### Increment 2 — 六库脚本 + Go migrate

- [ ] **Task 2.1** `taskAuth/run.sh` + `taskBill/run.sh` 增加 `migrate`；`main.go` 支持 `migrate` 参数  
- [ ] **Task 2.2** 其余 `db/*/migrate.sh` + `init.sh`（git-oauth, email, ai-provider, task-auth, task-bill）  
- [ ] **Task 2.3** `runAll/go.mod` 增加 `dbload` replace；`infrastructure/database_platform_reset.go`

### Increment 3 — Infra + runner

- [ ] **Task 3.1** `db/_infra/redis-flush.sh`, `db/_infra/kafka-recreate.sh`  
- [ ] **Task 3.2** `runner.ResetAllDatabases` + owner 停服映射  
  - 验证: `go test ./src/... -run ResetAll`

### Increment 4 — API / UI / YAML

- [ ] **Task 4.1** `ui.go` handler + `allowDevDatabaseReset`  
- [ ] **Task 4.2** `status.html` dev-tools 栏 + `resetAllDatabases()`  
- [ ] **Task 4.3** `ui_test.go` 片段断言；`value-stream.yaml` 追加 stream  
- [ ] **Task 4.4** `db/README.md` 开发重置章节  
  - 验证: `cd runAll && go test ./src/...`

## 执行顺序

Task 1.1 → 1.2 → 1.3 → 1.4 → 2.1 → 2.2 → 2.3 → 3.1 → 3.2 → 4.1 → 4.2 → 4.3 → 4.4

## 不自动启服（验收）

重置 API 返回成功后，`runner.processes` 中 saas-backend/task-auth 仍为空（用户手动启动）。
