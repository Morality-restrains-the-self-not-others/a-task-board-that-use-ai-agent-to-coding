# ADR-0012: runAll 健康检查端口必须等于进程监听 SSOT

- **Status:** accepted
- **Date:** 2026-08-16
- **Author:** Trae AI 团队
- **Deciders:** Trae AI 团队

---

## Context

task-events intent 消费者的监听端口 SSOT 在 `conf/events/domain-events/<event>/config.yaml`。`conf/runAll.yaml` 却手写 `health_check.url: http://${INFRA_HOST}:<port>/...`。新增 intent 时从相邻条目复制 YAML，容易只改 `name`/`start_command`、漏改端口。

2026-08-16：`2_fanout_work_panel_sse` 监听 18048，探活被写成 18056（`wechat-identity-conflict` 的端口），精准编译重启 `READINESS_TIMEOUT`。

平台服务已用 `conf_app` + `health_path` 避免第二份端口；task-events 没有等价约束。

## Decision

**We will** 规定 runAll 探活地址必须等于进程 listen SSOT，并用 CI 阻断漂移：

1. task-events：`runAll.yaml` health/liveness 端口、`intent_registry.go` `Port`、Prometheus file_sd target **全部等于** domain-events YAML `intents.<intent>.port`
2. 禁止两服务共用同一 health 端口；`url` 与 `liveness_url` 端口必须相同
3. 新的 domain-events `groupId` 必须有 runAll 条目（存量缺口只减不增 allowlist）
4. 平台服务继续用 `conf_app`，禁止再手写一份端口

不在本决策中改 runAll 加载器去「运行时覆盖错误 URL」（避免一次改 34 条编排语义）；靠提交门禁 + Agent 元规则防止再写入错误端口。

## Alternatives Considered

### Alternative 1: runAll 增加 `intent_path`，加载时从 YAML 拼 health URL

- **Pros:** 编排文件不再手写端口，从根上消灭拷贝错误
- **Cons:** 改 runAll schema 与 34 个服务条目；加载失败模式需单独测
- **Why rejected:** 先用 CI 堵住已知缺陷；schema 生成可作为后续 OPT

### Alternative 2: 仅文档/companion，不设 CI

- **Pros:** 改动面小
- **Cons:** Agent 仍会复制相邻 health_check
- **Why rejected:** 本次事故就是文档挡不住的拷贝

### Alternative 3: 立刻把 9 个未编入 runAll 的 intent 全部补进编排

- **Pros:** 覆盖率 100%，无需 allowlist
- **Cons:** 扩大本次治理范围（启动依赖、Kafka group、精准重启面）
- **Why rejected:** 与「防止将来再错端口」正交；由 OPT-20260816-052 跟踪

## Consequences

### Positive

- 提交阶段即可发现探活端口与 listen 不一致
- 新增 intent 漏写 runAll 会被门禁拦住（allowlist 外）

### Negative / Trade-offs

- runAll.yaml 仍手写端口，与 SSOT 双份
- 9 个存量 intent 暂不强制编入 runAll

### Mitigations

- 元规则检查清单：先 YAML `port`，再 registry，再 runAll URL
- allowlist `LEGACY_UNMANAGED_GROUP_IDS` 只减不增

## References

- `.ai/01_project_constraints/52_runall_health_port_ssot.md`
- `.cursor/rules/runall-health-port-ssot.mdc`
- `.ai/09_failure_experience/02_runtime_errors/94_task_events_fanout_health_port_mismatch.md`
- `db/scripts/ci/check_task_events_runall_health_ports.py`
