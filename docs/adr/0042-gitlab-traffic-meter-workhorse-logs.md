# ADR-0042: GitLab 出站流量以 workhorse 日志为计量 SSOT

- **Status:** accepted
- **Date:** 2026-08-25
- **Author:** cursor
- **Deciders:** /goal 自动采用

---

## Context

租户设置页 `gitlab-traffic-used-gb` 需要反映公网 `git-upload-pack` 出站字节。先前路径依赖任务容器在 `git clone --progress` 完成后 POST `received_bytes`。后续容器可能由第三方开发，**不会**上报该字段，计量会漏计。容器上报与 GitLab 侧再采集还会双计。

GitLab CE workhorse 已在 access / `git_traffic` 日志中写出 `uri`、`written_bytes`、`correlation_id`。Omnibus 日志已 bind-mount 到 `${GITLAB_HOME}/logs`。闸门（ADR-0036）只做 allow/deny，不写用量。

## Decision

We will **meter outbound Git traffic on the GitLab instance**, not in the task container.

1. **SSOT**：各区域 GitLab 旁路 sidecar 跟随 `gitlab-workhorse/current`，解析 HTTP access 中含 `git-upload-pack` / `git-upload-archive` 的行（以及带 `service` 的 `git_traffic` 且能解析出 `project_path` 的行），将 `written_bytes` POST 到 `taskBill` `POST /api/internal/taskbill/charge-gitlab-traffic/`。
2. **租户解析**：请求可只带 `project_path` + `region`（`TRAE_GITLAB_REGION`），复用闸门的 `tenant-{id}` 组路径与 GitLab 用户名归集；禁止把空 region 写成 `defaultGitlabRegion`。
3. **幂等**：`idempotency_key = wh:{correlation_id}`；同一行重放不双加。
4. **跳过语义与闸门对齐**：CI、配置的内网 Host、源 IP 为 10/8 或 192.168/16 不计；Docker NAT `172.16/12` **不计为内网**。
5. **删除** taskCloudService 容器 `received_bytes` 计量调用，避免双计。容器仍可上报进度，但用量不以该字段为准。
6. **禁止**在 taskBill / taskCloudService 业务进程内 ticker 扫日志（ADR-0011）。采集只允许 gitService compose sidecar（跟随日志写入）。
7. **不改** `gitService/gitlab-ce/` 源码树。

首次启动 sidecar 从当前 `current` 文件末尾开始（不回放历史日志），避免一次把存量 clone 全部计入预购配额。

## Alternatives Considered

### Alternative 1: 继续依赖容器 `received_bytes`

- **Pros:** 已落地；字节接近 clone 侧收到的 pack
- **Cons:** 第三方镜像可不报；笔记本/CI 外 clone 漏计
- **Why rejected:** 产品要求计量不依赖容器实现

### Alternative 2: 改 GitLab CE workhorse 同步 HTTP 回调

- **Pros:** 无日志延迟
- **Cons:** 必须维护 CE 补丁，升级成本高
- **Why rejected:** 禁止改 vendored CE；initializer + 日志 sidecar 已够用

### Alternative 3: taskEvents timer 扫宿主机日志

- **Pros:** 集中在事件进程
- **Cons:** 多区域 GitLab 日志不在 INFRA 本机（上海实例在 sh）；跨节点拉日志脆弱
- **Why rejected:** 采集必须跟 GitLab 实例同机

## Consequences

### Positive

- 第三方容器不报流量时，设置页仍能随 clone 增长
- 闸门与计量共用租户/region/内网语义
- 容器路径删除后无双计

### Negative / Trade-offs

- access `written_bytes` 含 HTTP 封装，略大于纯 pack
- sidecar 故障期间漏计（闸门仍可拦超额）；fail-open 与闸门一致不阻断 clone
- SSH git-upload-pack 若 gitlab-shell 无同等字节字段则暂漏（第三方容器主流为 HTTPS）

### Mitigations

- 幂等键来自 workhorse `correlation_id`
- 结构化日志 `gitlab_traffic_metered` 带 region / bytes / project_path
- SSH 字节采集单列 OPT，不阻塞 HTTP SSOT

## References

- [ADR-0036](0036-gitlab-traffic-quota-gitaccess-gate.md) GitAccess 闸门
- [ADR-0011](0011-no-service-internal-poll-loop.md) 禁止业务进程内轮询
- [ADR-0014](0014-pluggable-multi-region-gitlab.md) 多区域 GitLab
- OPT-20260822-015（本决策落地后关闭）
