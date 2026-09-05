# ADR-0048: 其它 runAll 服务不得 depends_on GitLab

- **Status:** accepted
- **Date:** 2026-08-27
- **Author:** Trae AI
- **Deciders:** Trae AI 团队

---

## Context

平台 GitLab 是可插拔组件（[ADR-0014](0014-pluggable-multi-region-gitlab.md)）：本机 `git-service` 默认不启动（[ADR-0047](0047-local-gitlab-runall-start-opt-in.md)），上海等区域实例由人工在 9999 面板对齐。

`conf/runAll.yaml` 曾让 `task-git-oauth` `depends_on: [git-service]`。runAll 在依赖为 `failed`/`skipped` 时会 **skip 下游**。本机 GitLab 被闸门拒绝后，OAuth 及依赖 OAuth 的网关、任务服务整条链被拖停——这把可插拔变成了启动硬依赖。

OAuth / OIDC / 计费对 GitLab 的调用是 **运行时 HTTP**，实例 URL 由人工对齐；GitLab 未就绪时应 fail-open 或返回业务错误，而不是阻止进程启动。

## Decision

We will **not** declare `depends_on: git-service` or `depends_on: git-service-*` on any runAll service except the GitLab instances themselves.

- GitLab entries remain in group `gitlab-regions` so operators can start/stop them from the panel after conf opt-in (local) or SSH (region). Other services start independently (`task-git-oauth` only waits on `docker-mysql`).
- Group `gitlab-regions` sets `skip_start_all: true`: one-click start-all does not start `git-service*`; 9999 panel single-service and group start still work.

Runtime clients (taskGitOauth, taskAuth OIDC, taskBill Admin API) keep their GitLab URLs in conf; they must tolerate GitLab being down.

## Alternatives Considered

### Alternative 1: 保留 task-git-oauth → git-service

- **Pros:** OAuth 启动时 GitLab 已探活
- **Cons:** 本机 GitLab 默认关闭后整条 platform 被 skip；停 GitLab 会要求先停 OAuth
- **Why rejected:** 与「可插拔、人工对齐」冲突

### Alternative 2: 从 runAll 删除全部 GitLab 条目

- **Pros:** 编排图更干净
- **Cons:** 偶发需要本机/区域实例时没有面板入口（ADR-0047 已拒绝）
- **Why rejected:** 条目保留；只去掉 **入向** 启动依赖

### Alternative 3: GitLab skip 时仍启动下游（改 runAll 语义）

- **Pros:** 不必改 YAML
- **Cons:** 改变全局 DAG 语义，影响所有 `on_failure: skip` 服务
- **Why rejected:** 局部错误的 depends_on 不应改编排器语义

## Consequences

### Positive

- start-all 在本机 GitLab 关闭时仍拉起 OAuth / 网关 / 任务链
- 停止 GitLab 不再级联停止 OAuth
- 编排图与产品模型一致：GitLab 可插拔

### Negative / Trade-offs

- OAuth 可能在 GitLab 未就绪时启动，首次授权会失败，需人工对齐后再试
- 新增服务时仍可能误把 `git-service` 写进 `depends_on`

### Mitigations

- `TestProductionConfig_NoInboundDependsOnGitLab` 阻断入向依赖
- `TestProductionConfig_StartAllPlanOmitsGitLabRegions` 锁定 start-all 不含 `git-service*`
- `conf/runAll.yaml` 与 companion 写明约束

## References

- [ADR-0014 可插拔多区域 GitLab](0014-pluggable-multi-region-gitlab.md)
- [ADR-0047 本机 GitLab 启动 opt-in](0047-local-gitlab-runall-start-opt-in.md)
- `conf/runAll.yaml` `gitlab-regions` / `task-git-oauth`
- `runAll/src/config_test.go` `TestProductionConfig_NoInboundDependsOnGitLab`
