# 任务详情硬件配置：仅启动配置，与运行态解耦

- **Status:** accepted
- **Date:** 2026-08-13
- **Iteration:** hardware-panel-startup-config-only
- **Based on:** application-integration current v75；enterprise-landscape current v13
- **Architecture impact:** **否** — 纯 taskFE 展示职责划分，不增删服务、数据流、接口
- **ADR:** No-ADR: trivial tech choice, no architectural impact
- **python_api_approval:** not_applicable（无新接口）

---

## 0. 问题

任务详情「环境与硬件」卡（`#hardware-config-section`，挂在评论区镜像下，意图 026）底部仍渲染 `HardwareServerControls`：启动 / 停止 / 跳转，以及按**任务级运行态**算出的禁用文案（「服务器已在运行，如需重新启动请先停止」）。

产品约束（2026-08-13 核对）：

> 这一区域**仅用来控制启动时的配置**，不关联任何运行中的状态；运行中的状态由**各个评论区域**自行关联。
>
> 改完后：**这边的硬件配置 = 服务于 `@镜像` 运行的硬件配置**（该卡是 `@镜像` 启机规格的 UI SSOT）。

因此硬件卡不得再根据 `isServerRunning` / `serverRuntimeStatus` 显隐或禁用任何控件。停止 / 跳转 / Workbench / 启动进度已在各评论「执行细节」的运行状态区（`comment-execution-runtime-with-start`）。

页面来源：`p[data-testid="start-server-disabled-reason"]`。

## 1. 决策（待批准后锁定）

**本卡是 `@镜像` 启机硬件的 UI 真源**，不是任务级「现在这台机器」的控制台。

1. **未 `@镜像` 时不展示镜像下拉、不展示硬件卡**（智能体资源配置一并仅在将运行时出现）。评论区默认只有输入框；提交文案为「提交评论」。
2. **评论中出现 `@镜像` 后**，自动展开硬件配置供调整（镜像已由 @ 选定，不再另放常驻「镜像」下拉）。提交文案为「提交并运行」。
3. **从硬件面板移除 `HardwareServerControls`**：无启动/停止/跳转/「已在运行」提示；UI 不读运行态。
4. **运行态只留在各评论执行细节**。
5. **有 `@镜像` 时按本卡当前规格启机**：项目模版摘要或临时配置覆盖（`server_run_template` 进评论 POST / 事件）。
6. 项目运行模版页、start-vm HTTP 路径不变。

### 1.1 配置如何被「启动」消费

| 硬件卡状态 | `@镜像` 启动所用模版 |
|------------|----------------------|
| 未展开临时配置（项目模版摘要） | 与现在相同：关联项目 `server_run_template` |
| 已展开临时配置且 payload 完整（有 cloud_platform_id + region） | 使用卡内 `buildRunTemplatePayload()` |
| 已展开但配置不完整 | 不拦截发评；composer 提示无法用该临时规格启动；不发残缺 override（启动回落项目模版） |

事件字段（可选）：`server_run_template`（对象，形状与项目运行模版相同）。**不落评论表**，只进 Kafka 事件，避免扩 DDL。

自动释放：若 payload 暂无该字段，沿用消费者/start-vm 既有默认；本迭代不新造 auto-release 契约。

### 1.2 明确非目标

| 非目标 | 理由 |
|--------|------|
| 把「启动服务器」迁到运行状态 Tab | 启动由评论提交触发 |
| 新服务 / 新架构版本 | 同一条 Rel_Flow，仅可选 JSON 字段 |
| 临时配置写入 `task_comments` 表 | 一次性启动参数，事件足够 |

## 2. 方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| 旧 C：运行中隐藏、停机仍显示启动 | 保留临时规格开机按钮 | 硬件卡仍绑定任务级运行态 | **废弃** |
| 只拆按钮、临时配置不进事件 | UI 干净 | 「启动配置」对 @镜像 不生效 | **废弃**（产品否定） |
| **A + 事件可选 `server_run_template`** | 卡内零运行态；临时规格真正用于下次 @镜像 启动 | 扩展评论 POST / 事件字段；Playwright 改锚 | **采用** |

## 3. 前端行为

### 3.0 评论区显隐（`pendingMentionId`）

| 状态 | 镜像下拉 | 环境与硬件卡 | 智能体资源配置 | 提交按钮 |
|------|----------|--------------|----------------|----------|
| 未 @镜像 | 不渲染 | 不渲染 | 不渲染 | 提交评论 |
| 已 @镜像 | 不渲染（镜像即 mention） | 自动出现，可调规格 | 出现（与运行绑定） | 提交并运行 |

Teleport 槽随 `pendingMentionId` 挂载；`defer` 保证槽出现后再送入。

### 3.1 组件

- `ServerConfigHardwarePanel`：去掉 `<HardwareServerControls>`；`runTemplateMode` 分支本来就不渲染，一并干净。
- `HardwareServerControls.vue`：任务详情不再引用。文件可暂留或删除；若删除须改 `hardwarePanelSplit.test.js`。
- `ServerConfig.logic.vue` / `HardwareConfigCommentBar.vue`：硬件面板不再传运行态 props，不再听 `@stop-server` / `@start-request-accepted`（硬件路径）。

### 3.2 显隐（无运行态条件）

硬件卡**始终**只显示配置；**任何**任务运行状态下都没有 `#start-server-btn` / `#stop-server-btn` / `#jump-to-server-btn` / `start-server-disabled-reason`。

### 3.3 composable 与评论提交

- `buildRunTemplatePayload` 保留并暴露。
- `startServer` 可暂留但 UI 不调用。
- `submitComment`：若面板 `hardwareConfigSource === 'temporary'` 且 payload 含 `cloud_platform_id`+`region`，写入 POST body `server_run_template`。
- taskTaskService：有 mentions 时把该对象拷进 `TASK_COMMENT_IMAGE_MENTIONED`（忽略无 mentions 的请求中的该字段）。
- taskEvents `1_start_vm_for_at_mention`：`payload.server_run_template` 为非空对象则用之，否则 `ResolveProjectServerRunTemplateFromProjects`。

### 3.4 测试锚点

- 单元：硬件壳源码不含 `HardwareServerControls`；mount 后无上述 testid。
- Playwright：`TaskDetail.start-vm-*.playwright.test.js` 改为走评论 `@镜像` / 既有 mock 启动路径，或标记改锚到评论执行细节（不得再点 `#start-server-btn`）。
- `TaskDetail.server-lifecycle-stop-vm`：停止只点运行状态区按钮。
- 遗留 `modal-task-detail.js`：找不到 `#start-server-btn` 时已有 warn，保持跳过。

## 4. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 硬件卡与运行态解耦（去按钮） | — | — | — | 纯前端显隐 |
| 评论 @镜像 启动（既有，payload 可带临时模版） | TASK_COMMENT_IMAGE_MENTIONED | taskTaskService 创建评论 | `1_start_vm_for_at_mention` → start-vm | 扩展可选字段，事件名不变 |

## 5. Domain Concept Inventory

- **Bounded Contexts**：Cloud Resource（既有）；taskFE 展示层将「启动配置」与「评论运行态」分成两个 UI 边界
- **Key Entities**：无新实体
- **Domain Events**：无新增

## 6. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | Nodes 108 / Edges 937 / Files 17；branch `main`；updated 2026-08-12T20:40:36 |
| 关键发现 | 图未索引 `HardwareServerControls` |
| 决策影响 | 源码爆炸半径：`ServerConfigHardwarePanel`、`HardwareConfigCommentBar`、`useServerConfigHardwarePanel`、若干 Playwright、`hardwarePanelSplit.test.js` |
| skip 理由 | CRG 可用但符号未入图，以源码检索补全 |

## 7. 价值流影响

- 展示侧：`task-detail-runtime-relay`（硬件卡不再承担启停）
- 测点：`TaskDetail.start-vm-*.playwright.test.js`、`TaskDetail.server-lifecycle-stop-vm.playwright.test.js`
- 不改 `<service>.<table>.<field>`；不新增 stream

## 8. 🏛️ 架构变更影响

- **不创建**新架构 target。同一条 `taskFE → taskTaskService → Kafka TASK_COMMENT_IMAGE_MENTIONED → taskEvents → taskCloudService start-vm`，仅事件/POST 增加可选字段。
- current：`v75-application-integration`（✅ shipped）、`v13-enterprise-landscape`（✅ current）
- 积压 target：v74 / v76 / v77（本需求不叠加）
- **ADR:** No-ADR: trivial tech choice, no architectural impact（既有事件可选字段，无新服务/新栈）

## 9. 实施计划（批准后）

1. 面板壳去掉 `HardwareServerControls`；停传运行态 props
2. 评论 POST + 事件透传 `server_run_template`；消费者优先覆盖
3. 单测：硬件卡无启停；handler 覆盖/回退；comment create 透传
4. Playwright：去 `#start-server-btn`；停止测例改评论运行区
5. Swagger：POST comments 可选 `server_run_template`
6. 登记精准编译重启：taskFE、taskTaskService、taskEvents
