# ADR-0014: 可插拔多区域 GitLab（独立 CE 实例 + 区域注册表）

- **Status:** accepted
- **Date:** 2026-08-18
- **Author:** cursor
- **Deciders:** 头脑风暴 `/1-brainstorming-design-docs` 用户批准

---

## Context

平台 GitLab 目前是单实例（`gitlab.${baseDomain}` → 边缘 nginx → `:8012`），`taskBill` 虽有 `billing_gitlab_region` 与 `(tenant_id, region)` 配额表，但开通路径仍使用全局 `GITLAB_API_BASE` / 默认 slug `tencent-shanghai-5`。业务需要在上海机（`Host sh`）新增一套 GitLab，并允许后续继续插拔更多区域；租户按区域选购磁盘/流量。

约束：

- 现网实例必须保留（不迁机）。
- 新域名 `gitlab-tencent-sh-1.${baseDomain}`。
- 上海机约 3.6G 内存、sshd 占用 2222。
- 新增接口默认落 Go（扩展 `taskBill` / `taskFE`，不新增 Python endpoint）。

## Decision

We will treat each platform GitLab **region as an independent GitLab CE instance** with isolated data.

We will use **two-layer SSOT**:

1. **商品/路由层**：`billing_gitlab_region`（slug、web URL、API base、admin token、容量、启停）。
2. **部署配方层**：`conf/infra/git-service-<slug>/config.yaml`（端口、内存、OIDC、GITLAB_HOME）。

We will **not** provide a platform default region: tenants must explicitly purchase one or more active regions. Purchase triggers **hybrid provisioning** (auto Admin API to that region's instance; failure → `pending` for human retry).

We will **not** make other runAll services wait on GitLab at start: no `depends_on: git-service*` except on the GitLab instances themselves. OAuth/OIDC/billing talk to GitLab over HTTP at runtime; operators align instances by hand ([ADR-0048](0048-gitlab-not-runall-start-dependency.md)).

Phase 1 instances:

| slug | web | notes |
|------|-----|--------|
| `tencent-shanghai-5` | `https://gitlab.${baseDomain}` | 现网，保留，可售 |
| `tencent-sh-1` | `https://gitlab-tencent-sh-1.${baseDomain}` | SH 本机 CE；HTTP :8014；SSH 宿主 2223；精简 ~2g |

## Alternatives Considered

### Alternative 1: 把现网 GitLab 迁到上海机（单实例搬家）

- **Pros:** 运维面简单
- **Cons:** 停机切流；与「可插拔多区域」目标冲突
- **Why rejected:** 用户改为保留现网并新增实例

### Alternative 2: 多区域共用同一 GitLab，仅用 group 路径区分

- **Pros:** 一台机器、一套 OIDC
- **Cons:** 故障域共享；无法按云区域隔离数据与合规
- **Why rejected:** 用户选择独立实例隔离

### Alternative 3: 平台默认区域 + grandfather 存量

- **Pros:** 存量无感
- **Cons:** 继续隐藏「必须选购」的产品模型
- **Why rejected:** 用户选择无默认；现网登记为可售区域 A，新租户从零选购

## Consequences

### Positive

- 后续新区域只需：部署实例 + 登记 region 行 + 边缘 vhost，不必改核心购买模型
- 开通打到错误实例的风险可通过「禁止全局 fallback」消除
- 现网与上海可独立扩缩容
- 其它平台服务的 runAll 启动不绑 GitLab（ADR-0048）

### Negative / Trade-offs

- 每实例独立 OIDC client、PAT、备份、探活
- 上海精简模式性能受限，可能 OOM
- 租户必须理解「选区域」；无默认增加设置页复杂度
- SSH clone 上海实例需非标端口 2223

### Mitigations

- 文档与 FE 空态引导选购
- 内存/OOM 监控 + 升配路径
- clone URL 展示 `advertiseSshPort=2223`
- 开通失败 `pending` + 管理后台重试

## References

- 设计文档: [2026-08-18-pluggable-multi-region-gitservice-design.md](../superpowers/specs/2026-08-18-pluggable-multi-region-gitservice-design.md)
- 架构: v85 target `docs/architecture/v85-*-20260818-1439-cursor.*`
- 相关表: `dataMigrate/taskBill/030_gitlab_region.sql`、`031_gitlab_region_capacity_and_token.sql`
- 各区域实例登录策略：[ADR-0016: 禁止自行注册，仅允许 SSO](0016-gitlab-sso-only-no-self-signup.md)
- 编排：其它服务不得 depends_on GitLab：[ADR-0048](0048-gitlab-not-runall-start-dependency.md)
