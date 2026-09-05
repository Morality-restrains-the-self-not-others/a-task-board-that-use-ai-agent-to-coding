# ADR-0047: 本机 GitLab 须 conf 显式打开后才能经 runAll 面板启动

- **Status:** accepted
- **Date:** 2026-08-27
- **Author:** Trae AI
- **Deciders:** Trae AI 团队

---

## Context

INFRA 本机 `git-service`（Docker 容器 `gitlab`，`:8012`）随 runAll「启动全部」或面板点启动时会拉起 Omnibus GitLab。Puma/Sidekiq 在配置错误（例如 `trusted_proxies` 中未加引号的 `::1`）时会崩溃重启，单实例即可打满约两核 CPU，且健康检查长期 unhealthy。

本机开发多数时间不需要这个 CE 实例（租户 Git 走区域 GitLab 或上海 `git-service-tencent-sh-1`）。把本机 GitLab 与其它服务一键拉起，成本高于收益。

## Decision

We will **默认禁止**本机 `git-service` 被 `runAll` 面板（`:9999`）或 `bash gitService/run.sh start` 启动。

SSOT 在 `conf/infra/git-service/config.yaml`：

```yaml
runAllStartEnabled: false
```

需要本机实例时，在 **gitignored** 的 `conf/infra/git-service/config.local.yaml` 覆盖：

```yaml
runAllStartEnabled: true
```

保存后从 runAll 面板启动 `git-service`。`stop` 不受此键限制。上海等区域实例（`GITSERVICE_CONF_APP != git-service`）不受此键约束。

闸门实现：`gitService/run.sh` 在 start/managed 路径读取 conf 导出的 `GITLAB_RUNALL_START_ENABLED`，非 `true` 则立即退出并打印改 conf 的说明。runAll 的 `start_command` 即该脚本，面板无需第二套开关。

## Alternatives Considered

### Alternative 1: 从 runAll.yaml 删除 git-service

- **Pros:** 面板上根本没有启动入口
- **Cons:** 偶发需要本机 GitLab 时要改编排 YAML 或走旁路 docker compose
- **Why rejected:** 仍希望「改一处 conf 即可从面板启动」，而不是拆掉编排条目

### Alternative 2: 仅跳过 start-all，面板单击仍可启动

- **Pros:** 误点「启动全部」不会拉起 GitLab
- **Cons:** 面板单击同样会空转占 CPU；与「必须先改 conf」的约定不一致
- **Why rejected:** 单击与 start-all 走同一 `run.sh start`，应同一闸门

### Alternative 3: 环境变量一次性覆盖、不写 conf

- **Pros:** 调试快
- **Cons:** 违反人工可改配置 SSOT（元规则 42）；容易被 shell 残留 env 意外打开
- **Why rejected:** 必须改 `conf/infra/git-service/` 下的 YAML

## Consequences

### Positive

- start-all / 面板默认不再拉起本机 GitLab，避免空转占 CPU
- 打开路径明确：改 `config.local.yaml` 后点 9999
- 停止与区域实例不受影响

### Negative / Trade-offs

- 需要本机 GitLab 的开发者多一步写 local 配置
- 有人把 `runAllStartEnabled: true` 提交进 `config.yaml` 会再次随面板启动（单测读取提交库断言必须为 false）
- 闸门拒绝会使 `git-service` 在 start-all 中 skip；其它服务不得因此被 skip（见 [ADR-0048](0048-gitlab-not-runall-start-dependency.md)）

### Mitigations

- 拒绝文案写明 `config.local.yaml` 与键名
- `test_load_gitservice_config.py` 断言提交库 `runAllStartEnabled: false`

## References

- `conf/infra/git-service/config.yaml` `runAllStartEnabled`
- `gitService/scripts/local_gitlab_start_gate.sh`
- [ADR-0014 可插拔多区域 GitLab](0014-pluggable-multi-region-gitlab.md)
- [ADR-0048 其它服务不得 depends_on GitLab](0048-gitlab-not-runall-start-dependency.md)
- 元规则 42：`.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md`
