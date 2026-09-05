# 可插拔多区域 gitService：保留现网 + 上海新增实例

- **日期**: 2026-08-18
- **作者**: cursor
- **迭代**: pluggable-multi-region-gitservice
- **状态**: accepted（总体设计已批准；架构 v85 target 已写入）
- **意图**: 待 `/4-value-stream` 切增量后补 `docs/intents/`
- **ADR**: [ADR-0014](../../adr/0014-pluggable-multi-region-gitlab.md)
- **python_api_approval**: n/a（零新增 Python 接口；落点 Go `taskBill` + Vue `taskFE` + 运维 `gitService`/`conf`/边缘 nginx）
- **架构**: **v85** 🎯 target（基于 v84 current）。每个视图四类伴生已生成。

---

## 0. 决策锁定（头脑风暴已确认）

| # | 决策 | 选择 |
|---|------|------|
| D1 | 部署形态 | **保留**现有 gitService；在 `Host sh`（`1.117.67.121`）**新增**一套独立 GitLab CE |
| D2 | 新域名 | `gitlab-tencent-sh-1.${baseDomain}`（默认 `gitlab-tencent-sh-1.daydaymoney.com`） |
| D3 | 数据隔离 | **每区域 = 独立 GitLab CE 实例**（不共享 DB / 仓库） |
| D4 | Phase 1 范围 | **完整可插拔**：管理员登记区域 + 租户选购一个或多个区域 |
| D5 | 平台默认区域 | **无默认**：租户必须显式选购至少一个区域才能使用平台 GitLab |
| D6 | 存量处理 | 现有 `gitlab.${baseDomain}` 登记为可售区域 A；**新租户从零选购**（不 grandfather） |
| D7 | 上海机内存 | **精简模式**（约 3.6G RAM，不升配） |
| D8 | 开通方式 | **混合**：购买后自动调区域 Admin API 建 group/配额；失败则 `pending` 人工介入 |
| D9 | 上海 SSH | 宿主机映射 **2223→22**（避开 sh 上 sshd 占用的 2222） |

---

## 1. 对当前架构的理解

根据 `docs/architecture/` current（v84）：

- **共有视图**: `enterprise-landscape`、`application-integration`（均为 v84 ✅ current）
- **业务层**: 计费/许可、项目与工作空间、身份
- **应用层**: `taskBill`（含 GitLab 区域/配额）、`taskGitOauth`、`taskFE` SystemAdmin、`gitService`（compose + run.sh）
- **技术层**: GitLab CE (:8012)、边缘 nginx（公网入口在上海机）、`INFRA_HOST` 默认 `10.2.150.68`

📋 架构版本历史（节选）：

- v84 (2026-08-16) ✅ current — 评论运行态推送同步
- v83 / v82 / v78 🎯 target — 正交积压（评论级仓库身份等）
- **本次迭代将在 v84 基础上设计 v85 target**

### 现状拓扑（已核实）

```
浏览器 / git clone
    │
    ▼
上海边缘 nginx (1.117.67.121 :443)
    │  server_name gitlab.daydaymoney.com
    │  upstream daydaymoney_gitlab → 127.0.0.1:8012
    ▼
反向隧道 / 现网 gitService（数据在 INFRA 节点 durable GITLAB_HOME）
    = 区域 seed: tencent-shanghai-5
```

上海机现状要点：

- 公网 DNS：`gitlab.daydaymoney.com` → `1.117.67.121`
- nginx 已有 `gitlab.daydaymoney.com` → `127.0.0.1:8012`
- **sshd 监听 2222**（管理 SSH），不可再给 GitLab shell 用 2222
- 内存 **~3.6G**；Docker 可用
- 本机尚未跑第二套 GitLab 容器

产品/代码已有「区域」骨架（**尚未真正多实例**）：

| 已有能力 | 缺口 |
|----------|------|
| 表 `billing_gitlab_region`（slug / api_base / web_url / token / 容量） | 仅 seed 一条 `tencent-shanghai-5`；无第二实例部署配方 |
| 表 `billing_tenant_gitlab_resource` PK `(tenant_id, region)` | 购买/展示路径仍大量 **硬编码默认** `tencent-shanghai-5` |
| SystemAdmin 区域 CRUD（`taskFE` + `taskBill`） | 缺「实例健康/端口/SSH advertise」等运维字段；创建区域 ≠ 拉起容器 |
| `ensureTenantGitlabGroup` | 只用**全局** `GITLAB_API_BASE` + 单 token，**不按 region 路由** |
| `conf/infra/git-service/config.yaml` | **单实例** SSOT；无多实例 conf 布局 |
| OIDC / redirect_uri | 绑定单一 `subdomains.gitlab` |

---

## 2. 🕸️ Code Review Graph 分析

- **CRG**: `.code-review-graph/graph.db` 存在（Nodes 108 / Files 17），索引面偏前端/脚本，**未覆盖** `taskBill` Go 符号。
- **结论**: `CRG unavailable for Go impact` — 爆炸半径以仓库 Grep 为准。

| 符号/路径 | 爆炸半径（Grep） | 设计动作 |
|-----------|------------------|----------|
| `defaultGitlabRegion` / `resolveRegionSlug` | `taskBill/src/gitlab_region.go` 及购买/配额 handler | 空 region → **400**，禁止静默默认 |
| `resolveGitlabAPIBase` / `resolveGitlabAdminToken` / `ensureTenantGitlabGroup` | `gitlab_disk_enforce.go`、`gitlab_disk_sync.go` | 改为 **按 region 解析** `billing_gitlab_region` |
| `SystemAdminGitlabResources.vue` | 区域 CRUD UI | 扩展字段：web URL、API base、SSH advertise、精简模式标记 |
| `conf/infra/git-service` + `gitService/run.sh` | runAll `git-service` 条目 | 保留现网实例；新增独立 conf 目录 + 远程启停约定 |
| 边缘 `/etc/nginx/conf.d/*` | 运维机（非仓内） | 新增 `gitlab-tencent-sh-1` server + upstream |

---

## 3. 问题陈述

平台要把「Git 托管」从**单点隐形默认**变成**可插拔商品**：

1. 运营可在不同云区域部署多套 GitLab CE；
2. 管理员在后台登记/启停区域（URL、API、token、容量）；
3. 租户按需购买一个或多个区域的磁盘/流量配额；
4. 业务开通（租户 group、配额）必须打到**正确实例**。

本期落地：保留现网区域 A；在腾讯上海机新增区域 B（`gitlab-tencent-sh-1`）；打通「登记 → 购买 → 自动开通（失败 pending）」闭环。

---

## 4. 选定方案（Target）

### 4.1 双实例拓扑

```
                    ┌─ HTTPS gitlab.daydaymoney.com ──────────────┐
浏览器/客户端 ──────┤                                           │
                    └─ HTTPS gitlab-tencent-sh-1.daydaymoney.com ─┤
                                                                ▼
                                              上海边缘 nginx (SH)
                                    ┌─────────────┴──────────────┐
                                    │ upstream :8012 (既有隧道)   │
                                    │ upstream :8014 (本机新容器) │
                                    └─────────────┬──────────────┘
                          ┌───────────────────────┴────────────────────┐
                          ▼                                            ▼
              现网 gitService (保留)                         新 gitService @ SH
              region: tencent-shanghai-5                    region: tencent-sh-1
              web: https://gitlab.${baseDomain}             web: https://gitlab-tencent-sh-1.${baseDomain}
              GITLAB_HOME: INFRA durable                    GITLAB_HOME: SH durable disk
              SSH advertise: 22 (既有链路)                   SSH host map: 2223→22
```

**端口约定（上海机）**

| 用途 | 端口 | 说明 |
|------|------|------|
| 现网 GitLab HTTP（隧道终点） | `127.0.0.1:8012` | 不变 |
| **新** GitLab HTTP | `0.0.0.0:8014`（或仅本机 + nginx） | 避开 8012 |
| 管理 SSH | `2222` | sshd，**禁止**给 GitLab |
| **新** GitLab SSH | `2223→22` | advertise 可用 `ssh://git@gitlab-tencent-sh-1…:2223/…` |
| HTTPS | `443` | 边缘终止 TLS |

### 4.2 可插拔模型（两层 SSOT）

| 层 | SSOT | 职责 |
|----|------|------|
| **商品/路由层** | DB `billing_gitlab_region` | 租户可见区域、API/Web URL、admin token、容量、启停 |
| **部署配方层** | `conf/infra/git-service-<slug>/config.yaml` | 该实例的 host/port/mem/OIDC/gitlabHome；由运维/Agent 部署 |

约定：

- **slug** 稳定主键：现网 `tencent-shanghai-5`；新实例 `tencent-sh-1`（与域名 `gitlab-tencent-sh-1` 对齐）。
- 新增区域时：**先**落部署配方并起容器，**再**在 SystemAdmin 写入/激活 `billing_gitlab_region`（或创建后 `is_active=0`，探活通过再激活）。
- **禁止**业务代码硬编码单一 `127.0.0.1:8012` 作为多区域真源；全局 env 仅作「开发单实例」fallback，生产路径必须带 `region`。

`conf/base.yaml`：

```yaml
subdomains:
  gitlab: gitlab.${baseDomain}                      # 现网（保留）
  gitlabTencentSh1: gitlab-tencent-sh-1.${baseDomain}  # 新增；后续更多区域按需加键或仅写在 region 表
```

> 域名硬编码仍只允许 `base.yaml`；区域表存**展开后的 URL**（管理员配置），不在业务 YAML 散落 FQDN。

### 4.3 无默认区域（产品规则）

- 所有租户侧 GitLab API：`region` **必填**；缺失 → `400 region_required`。
- 删除/停用 `defaultGitlabRegion` 静默回退（`resolveRegionSlug`、购买、配额、ensure group）。
- 租户未购买任何 `active` 区域配额时：设置页展示「请选购 GitLab 区域」，仓库创建/OAuth 平台 GitLab 入口不可用。
- 现网区域 `tencent-shanghai-5` **可售**；**不**自动给存量租户写配额（D6）。若存量租户已有 `billing_tenant_gitlab_resource` 行且 `region=tencent-shanghai-5`，视为已购该区域，行为不变。

### 4.4 购买 → 开通（混合）

```
租户选 region + 磁盘/流量 → taskBill 扣费写 billing_tenant_gitlab_resource
    → provisioning_status = provisioning
    → 异步/同步调用 ensureTenantGitlabGroupForRegion(region)
         成功 → active
         失败 → pending + 结构化日志 + 管理后台可见「待开通」
管理员可对 pending 重试或人工修正后标记 active
```

**强制改造**：`ensureTenantGitlabGroup` / 磁盘同步 / 限流执行改为：

```text
token = region.admin_private_token（空则拒绝并 pending）
api   = region.gitlab_api_base
```

禁止再读全局单例 token 去「猜」实例。

### 4.5 上海精简模式（D7）

新实例 conf 建议初值（可在 `conf/infra/git-service-tencent-sh-1/config.yaml` 调整）：

| 键 | 建议值 | 说明 |
|----|--------|------|
| `memLimit` | `2g` | 低于标准 4–8G；接受性能受限 |
| `pumaWorkers` | `1` | |
| `sidekiqConcurrency` | `2` | |
| `shmSize` | `256m` | |
| `minDockerMemoryMib` | `2048` | 触发精简路径 |
| `gitlabHome` | `/var/lib/daydaymoney/gitService-tencent-sh-1` | SH 非易失盘，禁止 tmpfs |
| `port` | `8014` | |
| `sshPort` | `2223` | 宿主机映射 |
| `advertiseSshPort` | `2223` | clone URL 展示 |

监控：容器 OOM / 502 → 告警；文档标明「升配至 ≥8G 后可调回标准参数」。

### 4.6 身份与 OIDC

每个 GitLab 实例独立 OmniAuth OIDC client：

| 实例 | redirect_uri（示意） |
|------|----------------------|
| 现网 | `https://gitlab.${baseDomain}/users/auth/openid_connect/callback` |
| 上海新 | `https://gitlab-tencent-sh-1.${baseDomain}/users/auth/openid_connect/callback` |

- `taskAuth` bootstrapClients **追加**新 client（或同 client 多 redirect，优先**独立 client_id** 便于吊销）。
- **交付缺口（2026-08-18）**：仅改 YAML 不够。`010_oidc_bootstrap_clients` 曾按 `data_migrate_log` 一次性 skip，导致 SH-1 client 未入库、SSO `client not found`。修复见 [oidc-bootstrap-client-seed-skip](./2026-08-18-oidc-bootstrap-client-seed-skip-design.md)。
- `gitService` 启动脚本按实例 conf 注入 `GITLAB_OIDC_*`。
- 登录/注册开关与现网一致：关闭手动注册与账密，仅 OIDC。

### 4.7 管理员后台

扩展现有 `SystemAdminGitlabResources`（非新 Django）：

- 区域列表：slug、名称、web URL、API base、cloud_provider、容量、allocated、is_active、精简模式标记、SSH advertise、最近探活。
- 创建/编辑：写入 `billing_gitlab_region`；token 写后脱敏展示。
- 动作：激活/停用、重试 pending 开通、（可选）「探活」`GET {api}/-/readiness` 或 `/users/sign_in`。
- **不**在 Phase 1 做「一键远程 docker compose up」——部署仍走运维脚本；后台只管**登记与商品化**。

### 4.8 租户侧

- 区域目录：列出 `is_active=1` 的区域（含价格/剩余容量提示）。
- 可对多个 region 分别购买磁盘/流量（已有 `(tenant_id, region)` PK）。
- 工作空间「代码仓库 / GitLab 连接」：必须选择**已购**区域；clone/Web 链接用该区域 `gitlab_web_url`。

---

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|----------|------------------|--------|--------------|---------|
| 管理员创建/更新 GitLab 区域 | GitlabRegionUpserted | taskBill SystemAdmin POST/PATCH | 审计；缓存失效 | 首期可 outbox 或结构化日志；建议补 Kafka 契约 |
| 管理员停用区域 | GitlabRegionDeactivated | taskBill | 租户侧不可再购；已购只读 | 同上 |
| 租户购买区域配额成功 | TenantGitlabResourcePurchased | taskBill POST | 触发区域开通 | 可与既有 BILLING_TRANSACTION_CREATED 并存 |
| 区域开通成功 | TenantGitlabRegionProvisioned | taskBill ensure 成功路径 | 审计；FE 刷新状态 | — |
| 区域开通失败转人工 | TenantGitlabRegionProvisioningPending | taskBill ensure 失败 | 通知管理员 | — |
| 列出区域 / 查配额 | — | — | — | 纯查询 |

禁止：购买成功后仅写库却不发开通意图、或开通时不带 `region` 打到错误实例。

---

## 6. Domain Concept Inventory（供 /6-ddd）

| 概念 | 说明 |
|------|------|
| **Bounded Context** | Billing（区域商品/配额）；GitHosting（GitLab 实例）；Identity（OIDC per instance） |
| **Entity** | `GitlabRegion`（slug）；`TenantGitlabResource`（tenant_id+region）；`GitLabInstance`（部署配方，非 DB） |
| **Aggregate** | Region（容量与 token）；TenantRegionQuota（购买与 provisioning_status） |
| **Domain Events** | 见上表 |
| **不变式** | 一 region ↔ 一 CE 实例；无租户默认 region；开通 API 必须使用该 region 的 api_base+token |

---

## 7. 价值流影响（输入给 /4-value-stream）

现有 `conf/value-stream.yaml` 已有 GitLab OIDC / external_url / 资源购买相关 stream。本需求影响：

| Stream / 步骤 | 影响 |
|---------------|------|
| 租户 GitLab 资源购买 | 强制选 region；多 region 配额行 |
| SystemAdmin GitLab 区域 | 第二实例登记；无默认 |
| GitLab OIDC SSO | 新实例独立 redirect / client |
| git-service 启动/探活 | runAll 仍管现网；SH 实例独立启停 + 边缘探活 |

预期新/改 fields（三段式）：

- `taskBill.billing_gitlab_region.gitlab_web_url`
- `taskBill.billing_gitlab_region.gitlab_api_base`
- `taskBill.billing_tenant_gitlab_resource.region`
- `taskBill.billing_tenant_gitlab_resource.provisioning_status`

---

## 8. 实施切片（建议顺序）

1. **Infra**：SH 精简部署 GitLab；DNS A 记录；nginx server+upstream:8014；SSH 2223；OIDC bootstrap。
2. **Conf**：`conf/infra/git-service-tencent-sh-1/` + `base.yaml` subdomain 键；现网 conf 不动。
3. **Billing**：去掉静默默认；`ensure*ForRegion`；购买后 hybrid 开通；seed/登记 `tencent-sh-1`。
4. **FE**：租户选购多区域；SystemAdmin 字段与 pending 重试；无区域时的空态。
5. **验收**：双域名可达；租户只购 sh-1 时 group 出现在新实例而非现网；现网租户行为回归。

---

## 9. 风险与缓解

| 风险 | 缓解 |
|------|------|
| SH 内存不足 OOM | 精简参数 + 监控 + 文档升配路径 |
| sshd:2222 与 GitLab SSH 冲突 | 固定 2223；文档写明 clone URL |
| 开通打错实例 | 单测强制 region token/base；禁止全局 fallback 在多 region 路径 |
| 无默认导致存量困惑 | 设置页文案；对已有 resource 行保持可用 |
| 边缘 nginx 在仓外 | runbook + 配置片段入库 `docs/` 或 `gitService/docs/edge-nginx-*.conf.example` |
| 双 OIDC client 漂移 | 每实例 conf SSOT + sync 脚本按 slug 执行 |

---

## 10. 非目标（本迭代不做）

- 一键远程编排（Kubernetes / 自动 docker 上架）
- 跨区域仓库复制 / 迁移工具
- Grandfather 批量赠送配额
- 把现网实例物理迁到 SH（明确保留）
- Python/Django 新接口

---

## 11. 🏛️ 架构变更影响

- **迭代版本**: v85 🎯 target
- **迭代名称**: pluggable-multi-region-gitservice
- **作者**: cursor
- **设计日期**: 2026-08-18 14:39
- **新增文件**（每个视图四类伴生格式，**缺一不可**）:
  - 🆕 `docs/architecture/v85-enterprise-landscape-20260818-1439-cursor.puml`
  - 🆕 `docs/architecture/v85-application-integration-20260818-1439-cursor.puml`
  - 🆕 `docs/architecture/v85-enterprise-landscape-20260818-1439-cursor.diff.archimate`（增量变迁：v84→v85 + Plateau/Gap/WP）
  - 🆕 `docs/architecture/v85-application-integration-20260818-1439-cursor.diff.archimate`
  - 🆕 `docs/architecture/v85-enterprise-landscape-20260818-1439-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v85-application-integration-20260818-1439-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v84-*-20260816-1328-cursor.puml` (current)
- **变更明细**: 🟢 GitLab CE SH-1 / `gitlab-tencent-sh-1` / conf 配方；🟡 taskBill 区域路由开通、taskFE 选购、taskAuth 多 OIDC、边缘 nginx；现网 GitLab 无标记保留
- **ADR**: [ADR-0014](../../adr/0014-pluggable-multi-region-gitlab.md)

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | 增量 — Plateau v84 → Gap（单实例默认）→ WP → Plateau v85；目标拓扑：FE→Bill→双 GitLab + 边缘分流 |
| **`.full.archimate`** | 全量 — 网关/FE/Bill/Auth/Cloud/SSE/GitOauth + 双 GitLab + 边缘 + 评论容器既有推送链 |

> 老文件未被修改。目标架构将在 `/10-ship` 执行时切换为 current。

---

## 12. 验收清单（设计级）

- [ ] `https://gitlab-tencent-sh-1.daydaymoney.com/users/sign_in` 200/302
- [ ] `https://gitlab.daydaymoney.com` 仍指向现网实例
- [ ] SystemAdmin 可见两区域；可停用其一
- [ ] 新租户不选区域无法「当作已有 GitLab」
- [ ] 仅购买 `tencent-sh-1` → group 只在新实例
- [ ] 开通失败 → `pending` + 可重试
- [ ] SSH：`ssh://git@gitlab-tencent-sh-1.daydaymoney.com:2223/...` 可达（精简实例）
- [ ] 无新增 Python HTTP 接口
