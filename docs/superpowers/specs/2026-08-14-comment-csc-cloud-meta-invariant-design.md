# 评论 CSC 云平台元数据不变量（禁止 mock / 空字段，允许覆盖模板）

- **日期**: 2026-08-14
- **作者**: cursor
- **迭代**: comment-csc-cloud-meta-invariant
- **状态**: accepted（2026-08-14 总体设计审批通过：模板仅新建默认；评论级可覆盖；非法字段才补齐）
- **ADR**: 不新增（不变量落在既有 `cloud_server_configs`；无新组件/协议）
- **意图**: `docs/intents/backend/cloud/comment_csc_cloud_meta_invariant.intent.md`

## 1. 问题背景

任务 `task_15805564155504060866` 库内实据：

| 行 | platform | region | authorization_id | instance_id |
|---|---|---|---|---|
| 任务级模板 `comment_id=''` | aliyun | cn-qingdao | cpa_-2740812859862113751 | （空，v80 正确） |
| 评论 CSC | **mock** | **空** | **空** | i-m5edps4oejpnwahrjpoa（真 ECS） |

`ensureCommentCloudServerConfig` 先写 `Platform: "mock"`，模板未就绪就落库；读路径 Workbench 报「暂不支持平台 mock」，runtime-status 因 mock 跳过租户授权。

上一轮修复用 `upgradeCommentCSCMetaFromTaskBase` **整段覆盖** platform/region/zone/auth/sg/vswitch。用户纠正：

> 评论 CSC 的 platform 不该是 mock，region 和 authorization_id 不该为空。任务级模板有数据，但用户仍可能在评论级改成别的。

因此模板不是持续 SSOT，只是**新建默认值**。

## 2. 对当前架构的理解（设计前确认）

根据 `docs/architecture/` current：

- 共有 2 个 current 视图：`enterprise-landscape` v80、`application-integration` v80
- 业务层：Cloud Resource Service / Task Management / IDE Workspace
- 应用层：`taskCloudService` 拥有 `cloud_server_configs`；评论行是运行实例权威，任务级行是模板 + 两计数
- 技术层：MySQL `task_cloud`（utf8mb4）
- 上次 shipped：v80 任务级两计数；相关 target：v78 评论级容器令牌

📋 架构版本历史（节选）：

- v80 (2026-08-14) ✅ current — 任务级仅模板+两计数；运行态只写评论 CSC
- v79 📦 archived — 厂商证照 COS
- v69 — 去掉本地 mock 启动；`comment_csc_bootstrap` 对 mock/空平台报错

本次**不新增架构版本**：无新服务、无新 HTTP 路径、无新表。只收紧 v80 已有 DataObject（评论 CSC）的字段不变量与覆盖规则。

## 3. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 评论 CSC 持久化后 `platform` 不是 `mock`、不是空 | ensure 单测；非法则不 INSERT |
| S2 | 评论 CSC 的 `region`、`authorization_id` 非空 | 同上 |
| S3 | 评论级合法三元组可以 ≠ 任务模板 | ensure/resolve/020 不覆盖非空合法字段 |
| S4 | 仅非法/空字段从模板补齐；不改 instance/server_url/public_ip | 单测 T4/T5 |
| S5 | 真实 `i-…` Workbench 走阿里云 URL；`mock-` 实例仍拒绝 | 既有 + T6/T7 |

## 4. 方案决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | 任务模板 = **新建默认值**，不是读时覆盖源 | 用户 / start-vm / 闲置复用可改评论级地域或授权 |
| D2 | 不变量：评论行 `platform` ∈ 真实云平台（当前 `aliyun`）；禁止 `mock`/`''`/`relay-local` | v69 已删本机 mock 启动 |
| D3 | 不变量：评论行 `region`、`authorization_id` 非空 | Workbench / Describe 都依赖这两项 |
| D4 | 补齐算法 **按字段**：只替换非法 platform、空 region、空 auth（及空 zone，若模板有） | 避免抹掉评论级覆盖 |
| D5 | 新建时补齐后仍非法 → **拒绝落库**，不再写 mock 占位行 | 根因是「先 mock 再等模板」 |
| D6 | 020 SQL 改为 `CASE WHEN` 空/mock 才取模板值 | 与 D4 一致；已跑过 020 的环境再跑幂等 |
| D7 | 不新增 path；不发 Kafka | 同库投影 / 校验失败无状态变更 |

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| 每次 resolve 把评论元数据重置成模板 | 抹掉用户改过的 region/auth |
| 继续默认 `platform=mock` 再升级 | 正是库内脏数据来源 |
| 评论永远只读模板、禁止覆盖 | 与「用户仍可能改成别的」冲突 |
| 新架构版本 / 新表 | 无组件与数据流增减 |

### 补齐规则（权威）

对评论行 `c`、任务模板 `t`（`comment_id=''`）：

```
if c.platform ∈ { '', 'mock', 'relay-local', relay* }:
    if t.platform 合法: c.platform = t.platform
    else: 非法（新建失败 / 读路径不覆盖成 mock）
if c.region == '':
    c.region = t.region   # t 空则仍空 → 非法
if c.authorization_id == '':
    c.authorization_id = t.authorization_id
if c.zone_id == '' 且 t.zone_id != '':
    c.zone_id = t.zone_id
# sg / vswitch：仅当评论为空时补；非空保留（评论可能挂不同网络）
# instance_id / server_url / public_ip / last_runtime_status：永不从模板拷
```

合法评论示例（必须保留）：

`platform=aliyun, region=cn-hangzhou, authorization_id=cpa-user`  
模板是 `cn-qingdao` + `cpa-template` → **不改评论**。

## 5. 数据模型

无新列。评论行语义：

```
comment_id = cmt_*
  运行权威: instance_id, last_runtime_status, public_ip, server_url
  云账号权威（可覆盖模板）: platform, region, zone_id, authorization_id, sg, vswitch
  不变量: platform 非 mock/空；region 非空；authorization_id 非空
```

任务级行继续只作默认模板 + `running_*_count`。

- 伸缩：L0（与 v80 相同；升级触发：单租户 CSC 年增量 > 100 万）
- 020 修订：只更新非法/空字段，utf8mb4 表已存在，无 DDL

## 6. 写路径 / 读路径

| 路径 | 行为 |
|------|------|
| `ensure` 新建 | 从模板填默认值；校验不变量；失败不 INSERT |
| `ensure` 已存在 | 只补非法/空字段并落库；合法覆盖不动 |
| `resolveScoped`（Workbench / runtime-status / stop） | 同上补齐；GET 可写回补齐字段（自愈脏行） |
| `PATCH server-config` | 今日只打任务级行；若带 `comment_id` 打评论行，须拒 mock/空 |
| start-vm / 闲置复用 | 写入评论级 region/auth 后视为用户覆盖，后续补齐不再改 |
| 020 | 与 ensure 同一 CASE WHEN |

## 7. API / 前端

- **不新增 path**。既有 `GET workbench-link` / `server-runtime-status` 在补齐后走真实阿里云。
- 路径已带 `tenant_id` + `workspace_id` + `task_id` + `comment_id`。分片键 **tenant_id** 合适。

## 8. 路径分片键审视（NFR 预览）

| 路径 | 已有 ID | 分片键判定 | 可伸缩性 | 动作 |
|------|---------|------------|----------|------|
| ensure / resolve 评论 CSC | tenant + workspace + task + comment | tenant 合适 | L1 | 无新 path |
| GET workbench-link | 同上 | tenant 合适 | L1 | 补齐后拼 URL |
| 020 UPDATE | 同库 JOIN | 非跨服务 | L0 | 幂等 CASE WHEN |

## 9. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 非法字段补齐并落库 | — | fill+upsert | — | 同服务同库投影 |
| 拒绝新建非法评论 CSC | — | ensure | — | 无状态变更 |

## 10. Domain Concept Inventory（供 /6-ddd）

| 概念 | 说明 |
|------|------|
| Bounded Context | 云资源（taskCloudService） |
| 实体 | `CommentCloudServer`（评论 CSC）；`TaskCloudServerTemplate`（任务级行） |
| 不变量 | CommentCloudServer 必须持有真实 Platform + Region + AuthorizationId |
| 覆盖 | 评论级三元组覆盖模板；模板仅 Default-on-Create |

## 11. 价值流影响

现有流：评论绑定 bootstrap、start-vm、Workbench、runtime-status。

- 影响字段：`cloud_server_configs.platform` / `region` / `authorization_id`（评论行语义：可覆盖、不可空）
- 测试：`comment_csc_ensure_test.go`（T1–T5）、`compute_scoped_csc_test.go`、`compute_workbench_link_test.go`
- 新流步骤建议：无；在既有 comment-csc / workbench 步加测试点
- 不新增跨流依赖

## 12. 🕸️ Code Review Graph 分析

- 图存在（`.code-review-graph/graph.db`）但 **0 nodes / 0 files**（从未 build）
- **CRG unavailable for blast radius**：以源码检索为准
- 已核对写点：`ensureCommentCloudServerConfig`、`applyTaskBaseMetaToCommentCSC`、`upgradeCommentCSCMetaFromTaskBase`、`healCommentCSCMockMetaFromTaskBase`、`resolveScopedCloudServerConfig`、`020_heal_comment_csc_mock_platform.sql`、`handlePatchServerConfig`（仅任务级）

## 13. 🏛️ 架构变更影响

- **不更新架构文件**（无新组件 / 数据流 / 基础设施）
- 语义补记：v80 评论 CSC 除运行态外，**云账号三元组为评论级可覆盖字段**，且禁止 mock/空
- 当前 current 保持 v80

## 14. 实施切片（批准后 /8-build）

1. Red：T1–T7（自定义覆盖不被抹；新建拒 mock；空字段才补）
2. 将 `upgradeCommentCSCMetaFromTaskBase` 改为按字段补齐；删除「默认 mock」
3. 修订 020 为 CASE WHEN
4. 调整上一轮「整段覆盖」测例，使其断言「只补非法字段」

## 15. 权限影响分析

不新增 HTTP 路径、不新增角色。补齐与校验均在已有工作区成员云资源读写上。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET workbench-link / runtime-status | workspace_member | Workspace | read | ensureTenantMember + comment_id | ✅ | — |
| POST ensure-client-ingress | workspace_member | Workspace | write SG | 同上；有 comment_id 走评论行 | ✅ | — |
| GET internal lookup | 内部服务 | Tenant | read | X-Internal-Secret | ✅ | — |
| ensure / 020 补齐 | 同租户同任务 | Resource | write 空字段 | 仅非法/空 | ✅ | 不覆盖用户合法三元组 |

评级绿灯。`python_api_approval`: 未触发（无新 Python 接口）。

## 16. 风险

| 风险 | 缓解 |
|------|------|
| 已执行旧 020 的行已被整段覆盖 | 合法覆盖若当时 platform=mock 已被写成模板值，无法自动还原；仅防今后再覆盖 |
| 模板也空 | 新建失败，前端/绑定走既有「未配置云平台」 |
| 旧进程仍写 mock | 精准编译重启 task-cloud-service |
