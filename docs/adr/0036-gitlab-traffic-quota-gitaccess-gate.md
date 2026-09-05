# ADR-0036: GitLab 出站流量配额在 GitAccess 闸门强制执行

- **Status:** accepted
- **Date:** 2026-08-23
- **Author:** cursor
- **Deciders:** /goal 自动采用

---

## Context

租户设置页 `gitlab-traffic-used-gb` 展示「已用 / 预购」。GitLab CE **没有**租户级出站流量/带宽硬限额（磁盘已有 `repository_size_limit`）。平台原先只把用量写入 `traffic_used_gb`（且用磁盘占用当流量下限），**不拦截** `git-upload-pack`。因此预购为 0（「未预购」）时，公网 clone/pull 仍可发生，用量还会随磁盘刷新被抬高。

区域卡片上的 `total_bandwidth_mbps` 只是运维录入的共享带宽包元数据，不是 Linux tc / GitLab 限速。

## Decision

We will **deny public git download** (`git-upload-pack` / `git-upload-archive`) when prepaid traffic is exhausted or never purchased.

1. **taskBill** 提供内部闸门 `POST /api/internal/taskbill/gitlab-traffic-gate/`，按 `(tenant, region)` 判定 `allowed`。
2. **gitService** 用既有 Omnibus initializer 注入模式（与 OIDC 补丁相同）`prepend Gitlab::GitAccess`，在 `check_download_access!` 之后调用闸门。
3. **GitLab CI job token** 与 **同区域内网**（请求 Host 为内网域名/GitLab 私网 IP，或源 IP 为 10/8、192.168/16）放行。任务贴走**公网 GitLab Host** 的克隆一律走配额（Docker NAT 的 172.16/12、Workhorse 127.0.0.1 **不**视为内网）。
4. **磁盘占用不再持续写入流量水位**：已用流量只来自计量计数与扣费流水；存量水位不降低。
5. 闸门不可达时 **fail-open**（打 warn），避免 GitLab 与 taskBill 短暂断连导致全员无法拉代码。

不新增服务、不改 GitLab CE 源码树。

## Alternatives Considered

### Alternative 1: nginx auth_request 拦 HTTP git

- **Pros:** 不碰 Rails
- **Cons:** 不覆盖 SSH；多区域 nginx 定制面更大
- **Why rejected:** GitAccess 同时覆盖 HTTP 与 SSH

### Alternative 2: 云厂商共享带宽包 / tc 限速

- **Pros:** 真限带宽
- **Cons:** 限的是整实例出口，不是租户预购 GB；无法表达「未预购则禁止 clone」
- **Why rejected:** 产品要的是配额耗尽后停止出站 Git，不是 Mbps 整形

### Alternative 3: 把项目 archive / 关掉 repository_access_level

- **Pros:** 纯 GitLab API
- **Cons:** 同时阻断 push 与 Web 浏览，误伤磁盘配额产品
- **Why rejected:** 流量闸门只应拦 download

## Consequences

### Positive

- 未预购或超额后公网 clone/pull 被拒绝，用量不再因 clone 增长
- 磁盘下限不再把 push 体积当成流量持续抬高

### Negative / Trade-offs

- GitLab Rails 每次 download 检查多一次 HTTP（15s 缓存缓解）
- 未映射到 `tenant-{id}` 组的仓 fail-open，仍可能漏拦
- 出站字节计量 SSOT 见 [ADR-0042](0042-gitlab-traffic-meter-workhorse-logs.md)（workhorse 日志 sidecar；不再依赖容器 `received_bytes`）

### Mitigations

- 租户组路径 `tenant-{id}` 作为闸门主键（与磁盘硬限同一约定）
- 设置页展示阻断原因与购买入口
- fail-open 打结构化 `gitlab_traffic_gate_unreachable`

## References

- [ADR-0014](0014-pluggable-multi-region-gitlab.md) 多区域 GitLab
- 磁盘硬限：`docs/superpowers/specs/2026-07-18-gitlab-disk-usage-quota-enforcement-design.md`
- 设计：`docs/superpowers/specs/2026-08-23-gitlab-traffic-quota-enforcement-design.md`
