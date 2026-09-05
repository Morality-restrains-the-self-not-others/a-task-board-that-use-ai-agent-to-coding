# 切换镜像时硬件配置绑定一致性 — 设计文档

- **Date:** 2026-08-21
- **Status:** accepted（2026-08-21 设计审批：硬拦截）
- **Author:** cursor
- **入口:** 项目详情默认镜像 + 运行模版；任务详情评论 `@镜像` + 环境与硬件卡
- **TraceId:** 无（静态代码审查；用户未提供 `data-traceId`）
- **python_api_approval:** 未触发（无新增 Python 接口）

## 架构理解（current 基线）

根据 `docs/architecture/` 与 `VERSION_HISTORY.md`：

- **current:** v92（2026-08-20 21:40）— 多渠道推荐码与分渠道分账；与本问题正交
- **积压 target:** v90（SaaS inbound skill 版本）、v89（ztree 日志持久化）
- **与本需求相关的既有组件（未改拓扑）:**
  - 业务层：项目配置默认运行环境、任务评论启动云主机
  - 应用层：`taskFE`（项目/任务 UI）、`taskProjectService`（`projects.container_image_id` / `server_run_template`）、`taskTaskService`（`tasks.container_image_id`）、`taskCloudService`（start-vm 解析宿主机镜像）、`taskAiProvider`（已安装容器镜像目录）
  - 技术层：云厂商 ECS 实例规格 + 厂商云主机镜像（runtime env）

本次是**既有绑定语义的缺陷修复**，不新增服务、不改数据所有权、不改 Kafka 拓扑。**不创建新架构 target 文件。**

## 🔍 Trace 日志分析

无 traceId。判定依据为源码与既有 Playwright/单测，非一次线上报错回放。

## 🕸️ Code Review Graph 分析

`CRG unavailable: .code-review-graph/graph.db 不存在；MCP 无 codegraph 工具。`

手工爆炸半径（同文件 → 同服务 → 跨语言）：

| 层 | 符号 / 文件 | 影响 |
|----|-------------|------|
| 项目页 | `ProjectDetailInlineEditableFields` PATCH `container_image_id`；`ProjectRunTemplatePanel` | 换镜像不 watch、不重绑模版 |
| 硬件面板 | `useServerConfigHardwarePanel.loadRegions` / `watch(selectedImageId)` | `runTemplateMode` 且模版已 apply 时跳过拉实例 |
| 架构推导 | `resolvePrimaryImageArchitecture` | 优先陈旧 `task.container_image`，忽略当前 `selectedImageId` |
| 任务镜像 | `useServerConfigImages` watch | 只 PATCH `container_image_id`，不碰硬件 |
| start-vm | `inferInstanceArchitecture` / `pickRuntimeEnvByArchitecture` | 后端最后防线；`r6` 子串会把 x86 `ecs.r6.*` 判成 arm64 |

同类问题搜索：`selectedImageId` + `container_image_id` + `server_run_template` + `inferInstanceArchitecture`。厂商门户 `VendorPortal.onImageChange` 是**云主机镜像表单**另一路径，不在本次范围。

## 结论（先回答问题）

**会有问题。** 容器镜像与硬件（实例规格 / 项目 `server_run_template`）是两条独立绑定。切换镜像**不会**自动按新镜像 CPU 架构重绑或失效已保存的实例规格。同架构换镜像通常无感；**跨架构**（x86_64 ↔ arm64）会把旧规格带到自动运行 / start-vm，最终由后端报「实例规格与宿主机镜像架构不匹配」，或更糟：`inferInstanceArchitecture` 误判时选错宿主机镜像。

## 问题分析

### 不变量（产品意图，来自 026）

硬件选型必须依赖**当前选中镜像**的 CPU 架构。项目运行模版与任务临时配置都通过同一 `ServerConfigHardwarePanel`。

### 缺陷 1 — 项目详情（用户点名场景）

1. 默认镜像 PATCH 只写 `container_image_id`（`ProjectDetailInlineEditableFields`）。
2. 硬件模版是独立字段 `server_run_template`。
3. `ProjectRunTemplatePanel` 把 `:selected-image-id="projectContainerImageId"` 传给面板，但**没有 watch 镜像 ID** 去清实例、重拉列表或提示模版失效。
4. `loadRegions` 在 `runTemplateMode && projectRunTemplateApplied` 时**跳过** `scheduleFetchAvailableInstances`。项目页模版一旦 apply，换镜像不会按新架构刷新实例；旧 `selected_instance` 仍绑在面板与已保存 JSON 上。
5. 自动运行 / 未打开硬件面板的创建任务路径会直接使用**旧实例类型 + 新镜像**。

### 缺陷 2 — 任务详情

`resolvePrimaryImageArchitecture` **优先** `task.container_image.target_architectures`，再才看 `selectedImageId` 对应的已安装镜像。单测 `containerImageArchitecture.test.js` 把「选了 x86 镜像仍返回 task 上的 arm64」写成正确行为。换镜像后 PATCH 完成前（或嵌套对象从未水合/已过期），实例列表仍按旧架构过滤，start-vm 却带新 `container_image_id`。

`useServerConfigImages` 只 PATCH `container_image_id`，不校验/清理硬件选中项。未展开临时配置时 `useProjectTemplateForStart` 用**项目模版硬件 + 评论新镜像**，前端无架构校验。

### 缺陷 3 — `loadRegions` 并发合并

`if (loadRegionsPromise) return loadRegionsPromise` 不区分 imageId。快速连换镜像会沿用上一张镜像的地域请求。

### 缺陷 4 — `inferInstanceArchitecture`

`strings.Contains(t, "r6")` 会把阿里云 x86 内存型 `ecs.r6.*` 判成 arm64（真 arm 规格应是 `r6r` / `g6r` / `c6r` / `g8y` 等）。这是 start-vm 架构匹配的独立假阳性/假阴性源，与换镜像叠加时更难排查。

### 后端已有防线（保留）

`pickRuntimeEnvByArchitecture` 在 desiredArch 与 runtime env 不一致时返回错误。这是最后防线，**不能**替代前端绑定与项目模版持久化一致性。

## 目标（完成标准）

1. 切换容器镜像后，硬件面板用于过滤实例的架构 = **当前 `selectedImageId` 在已安装目录中的 `target_architectures[0]`**（无目录命中才回退 task 嵌套对象）。
2. **硬拦截（已批准）**：项目 PATCH 若变更 `container_image_id`，生效模版（请求体 `server_run_template` 否则库中现模版）必须是**完整且与新镜像架构匹配**的运行模版，否则 **400、整单不写**。不允许先改镜像再稍后补硬件。
3. 架构仍兼容且模版完整时，可只 PATCH 镜像、不必重交模版（同 ISA 换镜像不打扰）。
4. `loadRegions` 按 imageId 代际，过期 promise 不得覆盖新镜像地域。
5. `inferInstanceArchitecture` 不再用裸 `r6` 子串；补单测覆盖 `ecs.r6.xlarge`（x86）与 `ecs.r6r.xlarge`（arm）。
6. start-vm 架构不匹配错误保留。

## 选定方案

| 方案 | 结论 |
|------|------|
| A. 仅前端 banner，不改 DB | 拒：自动运行仍带旧规格 |
| C. 换镜像后剥离不兼容实例、允许镜像先落库 | 拒（审批否决）：会出现「镜像已换、模版缺实例」窗口 |
| **B. 换镜像硬拦截：同次提交匹配的完整硬件模版，否则 400（采用）** | 跨架构必须先选好实例再保存镜像；同架构可只改镜像 |

### B 的行为（已批准）

**判定「兼容」：** 规范化后的实例架构 ∈ 新镜像 `target_architectures`。实例架构优先用面板/云 API 的 `architecture`；服务端用与 `inferInstanceArchitecture` **同一套收紧后启发式**（禁止裸 `r6`）。镜像架构来自已安装目录（`lookupInstalledImage` 已有 internal lookup，含 `target_architectures`）。**陈旧 `installed_image_id`（目录 404）不得用「无法查询架构」挡住同架构保存**：lookup 可带 `name=`，租户下该名唯一时回退现网镜像并愈合项目绑定；多名冲突或无名则 400「不存在或已卸载」，仍须同时写出镜像要求与实例支持的 ISA。

**判定「完整模版」：** 与现网 `projectHasConfiguredRunTemplate` / `buildAutoRunStartVmRequest` 对齐：至少有云平台、地域、以及可解析的 `instance_type`（`selected_instance` 或 `hardware_config.instance_type`）。缺实例视为不完整，换镜像时 400。

**项目 PATCH（`taskProjectService`，扩展既有接口，无新 path）**

1. **先校验再写**：当前 handler 对字段逐条 `Exec` 且无事务。硬拦截必须在任何 `UPDATE` 之前算好「生效镜像 + 生效模版」，失败则 400，镜像与模版都不改。
2. 若 body 含新的 `container_image_id`（与库中不同，含清空）：
   - 生效模版 = body 中的 `server_run_template`（若有）否则库中现模版。
   - 新镜像能解析到非空 `target_architectures` 时：生效模版必须完整 **且** 实例架构落在该集合内，否则 400。
   - 文案示例：`镜像要求的 CPU 架构（arm64）与实例系统支持的 CPU 架构（x86_64（实例规格 ecs.g7.xlarge））不匹配，请在同一次保存中提交完整且匹配的 server_run_template`。无法解析时须同时写出镜像要求与实例支持的 ISA。带 `data-traceId`。
   - 镜像架构解析失败（lookup 5xx）：**不得静默放行**；400 或 502，说明无法校验，避免跨架构漏网。
   - 镜像声明架构为空：400（无法证明匹配）；与「未知则放行」相反，硬拦截优先。
3. 同架构且现模版已完整匹配：允许只 PATCH `container_image_id`。
4. 仅 PATCH `server_run_template`、不改镜像：新模版也须与**当前**项目镜像匹配（举一反三，避免先存坏模版再换镜像绕过）。清空模版（`{}`）保持现网「未设置」语义，不因本规则 400。
5. Swagger：既有项目 PATCH 的 400 说明补本错误；不新增 Python 接口。

**项目详情 UI**

1. 默认镜像保存不得再发「仅 `container_image_id`」若当前面板/已存模版对新镜像不完整或不匹配。
2. 交互：用户改选镜像后，硬件面板按新架构拉实例并清空不兼容选中；保存镜像时 **同请求带上** 面板当前完整模版（用户须先选好实例）。未选好则禁用保存或保存后展示 400。
3. `loadRegions` 带 imageId 代际；`runTemplateMode` 换镜像也要拉 available-instances。
4. 防重放：镜像保存与「保存运行模版」均为写操作，沿用现有 busy/Idempotency 约定。

**任务详情**

任务 PATCH 只有 `container_image_id`、不存硬件。硬拦截落在**启动路径**而非任务 PATCH：

1. 修正 `resolvePrimaryImageArchitecture`：已安装目录命中时以 `selectedImageId` 为准。
2. 评论 POST 若带 `server_run_template`，须与该评论镜像匹配，否则 400（`taskTaskService` 或 start-vm 入口，优先 start-vm 已有校验 + 前端拦截）。
3. 不带临时模版、回落项目模版时：项目模版须与所选镜像匹配，否则禁止 start，提示到项目页改模版或展开临时配置并提交匹配规格。
4. `@镜像` 下拉可先改本地选中；在模版不匹配时 **不要** 让用户误以为已能跑（横幅 + 禁用启动）。是否立即 PATCH 任务镜像：允许 PATCH（与项目硬拦截不同），但启动硬拦。

**Go `inferInstanceArchitecture`**

- arm 信号改为明确族：`arm`、`g8y`/`c8y`/`r8y`、`c6r`/`g6r`/`r6r` 等，**禁止**单独 `Contains("r6")`。
- 未知规格保持默认 `x86_64`，由 `pickRuntimeEnvByArchitecture` 再拦一层。
- 项目 PATCH 与 start-vm **共用**该函数（可抽到 `shareLib` 或 cloud 侧校验；禁止两套启发式）。优先：校验逻辑放 `taskCloudService` internal API，`taskProjectService` 调用；避免 project 服务复制规格表。若为减少同步调用，允许把同一纯函数拷到 project 包并单测锁定，注释指向 SSOT 文件。 **推荐** 纯函数放 `taskCloudService` 并在 project 用 internal lookup 拿镜像架构 + 本地同名纯函数副本，CI 用共享 fixture 防漂移。

### 非目标

- 不把镜像与硬件合成单一聚合根 / 不改表结构。
- 不自动挑选「最接近」的另一架构实例（避免静默换规格与费用）。
- **不**在换镜像成功后静默剥离实例（方案 C 已否决）。
- 不改厂商门户云主机镜像表单。
- 不新增 Python 接口；不新增 Kafka 事件。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 项目切换默认容器镜像 | — | `taskProjectService` 既有 PATCH（可能同单写模版） | 校验失败 400 不落库 | 无对应新事件：同步配置写；失败不改变系统事实 |
| 项目保存运行模版 | — | 既有 PATCH | 须与当前镜像匹配否则 400 | 无对应新事件：同上 |
| 任务切换默认容器镜像 | — | `taskTaskService` 既有 PATCH | 只更新镜像 ID；启动另拦 | 无对应新事件：配置写 |
| start-vm 架构不匹配 | — | `taskCloudService` 同步错误 | 前端展示 / binding failed | 无对应新事件：同步校验失败 |

## Domain Concept Inventory（给 /6-ddd 的轻量输入）

- **Bounded Contexts:** 项目配置（Project）、任务协作（Task）、云资源供给（Cloud）
- **Key Entities:** Project（`container_image_id` + `server_run_template`）、Task（`container_image_id`）、InstalledContainerImage（`target_architectures`）、CommentContainerBinding / start-vm 请求
- **Candidate Aggregates:** Project 为镜像+运行模版的一致性边界（跨架构变更须同单提交匹配组合，否则 400）；Cloud start-vm 为运行时校验边界
- **Domain Events:** 本修复不新增

## 价值流影响

来源：`conf/value-stream.yaml`

| Stream / step | 影响 |
|---------------|------|
| 项目默认镜像 PATCH | 跨架构必须同单提交完整匹配模版，否则 400 |
| `create-task-auto-run-backend-start`（`projects.server_run_template`） | 硬拦截保证库中不会出现「新镜像 + 旧架构实例」；auto_run 不再依赖 start-vm 才发现 |
| 任务协作 / 评论 `@镜像` 启动 | 前端架构过滤与项目模版回落须一致 |

不新增 value stream。字段仍为三元组：`task-project-service.projects.container_image_id`、`task-project-service.projects.server_run_template`、`task-task-service.tasks.container_image_id`。JSON 内嵌实例键在描述中说明，不拆成四段路径。

预期测试：

| 类型 | 文件 |
|------|------|
| 单测 | `containerImageArchitecture.test.js`（翻转优先序） |
| 单测 | `useServerConfigHardwarePanel` / 抽出的「实例 vs 镜像架构」纯函数 |
| 单测 | `taskCloudService` `inferInstanceArchitecture`（新 `*_test.go`） |
| 单测 | `taskProjectService`：跨架构只 PATCH 镜像 → 400 且 DB 未改；同单带匹配完整模版 → 200；同架构只 PATCH 镜像 → 200 |
| Playwright | 扩展 `ProjectDetail.image-architecture-filter.playwright.test.js`、`hardware-config-sync`：跨架构未重选实例时保存镜像失败；重选后同单成功 |
| Playwright | 任务详情：换 `@镜像` 后实例列表按新架构过滤（补 `TaskDetail` 现有 hardware 测） |

## 权限影响分析

见 `docs/superpowers/specs/2026-08-21-image-switch-hardware-binding-permission-analysis.md`。无新角色；既有项目 PATCH 鉴权充分。

## 🏛️ 架构变更影响

- **判定:** 纯 Bug 修复 / 既有字段语义收紧 → **不更新** `docs/architecture/` target
- **current 保持:** v92
- **不新增:** `.puml` / `.diff.archimate` / `.full.archimate` / `.mermaid.md`

## 实施切片（批准后）

1. 抽出并单测 `instanceCompatibleWithImageArchitectures` + 收紧 `inferInstanceArchitecture`（共享 fixture）。
2. 修 `resolvePrimaryImageArchitecture` + 翻转单测。
3. 硬件面板：换镜像必拉实例；generation；不兼容清空；镜像保存拼上完整模版。
4. `taskProjectService` PATCH：先校验再写；400 契约单测；Swagger 4xx 文案。
5. 启动路径：项目模版与所选镜像不匹配则禁止 start。
6. Playwright：跨架构保存镜像失败 / 同单成功。

## 风险

- 用户只改镜像、未打开硬件面板：保存会 400，必须到运行模版选实例。文案要写清。
- lookup 已安装镜像失败时拒绝保存，可能误伤（cloud 短暂不可用）；属硬拦截代价，应返回可重试错误而非写库。
- 双架构镜像：实例架构落在集合内即通过。
- 存量已不一致数据：下次 PATCH 镜像或模版时会被 400 挡住，直到用户提交匹配组合；只读浏览不受影响。
