# ADR-0002: 应用进程独立于数据库迁移

- **Status:** accepted
- **Date:** 2026-08-03
- **Author:** Trae AI 团队
- **Deciders:** Trae AI 团队
- **Supersedes (partial):** ADR-0001 §「服务启动时通过 runDataMigrate() 执行」— 存放位置仍以 ADR-0001 / dataMigrate 为准；**执行权**改由本 ADR 约束

---

## Context

ADR-0001 已要求 DDL 集中在 `dataMigrate/`，但执行仍允许（并曾要求）业务进程启动时 `runDataMigrate()`。与此同时，runAll `http://<host>:9999/`「初始化全部数据库」已能通过 `db/registry.yaml` + `apply_datamigrate.sh` 独立完成全库迁移。双轨导致耦合、重复执行与启动 crash loop（如 DELIMITER / 半迁移状态）。

## Decision

**We will** 将数据库迁移与业务应用进程生命周期解耦：

1. Schema/seed **只**存在于 `dataMigrate/<service>/`
2. **只**通过 9999 初始化数据库（及其所调用的 `migrate.sh` / 显式 migrate CLI / 测试夹具）执行迁移
3. 长期运行的业务进程 **不**在启动或运行路径执行迁移

## Alternatives Considered

### Alternative 1: 保留启动时自动迁移 + 9999 双轨

- **Pros:** 开发机「启服务即建表」方便
- **Cons:** 隐式 DDL、难编排、失败拖垮服务、与统一入口冲突
- **Why rejected:** 正确性与可运维性优先于启服便利

### Alternative 2: 仅文档禁止、代码暂不改

- **Pros:** 改动面小
- **Cons:** 元规则无法落地，回归必然
- **Why rejected:** 与「禁止忽略」元规则目标不符

## Consequences

### Positive

- 初始化可在不启动业务的情况下完成
- 迁移失败不进入业务 crash loop
- 单一编排入口，审计清晰

### Negative / Trade-offs

- 开发流程必须先 9999 init（或 migrate.sh）再启服务
- Go seed（如 taskAuth OIDC bootstrap）须挂在 migrate CLI / migrate.sh，不能依赖 server 启动

## References

- `.ai/01_project_constraints/40_app_process_independent_of_db_migrate.md`
- `.cursor/rules/app-process-independent-db-migrate.mdc`
- `db/scripts/apply_datamigrate.sh`、`runAll` `InitAllDatabases`
