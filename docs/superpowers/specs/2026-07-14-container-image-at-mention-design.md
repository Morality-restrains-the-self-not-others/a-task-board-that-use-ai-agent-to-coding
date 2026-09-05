# 设计：工作空间容器镜像 `@` 模式 + 统一评论区

- **日期**: 2026-07-14 23:58
- **作者**: claude
- **状态**: approved（goal-mode 自动采纳，2026-07-15 00:10）
- **迭代名**: `container-image-at-mention`
- **相关页面**:
  - 设置：`/tenant/{t}/settings/task-panel/`
  - 任务详情：`/tenant/{t}/workspace/{ws}/task-detail/{task_id}/`
- **架构版本**: v26 🎯 target（基于 v25 current；与 v22/v23/v24 并行 target）
- **python_api_approval**: n/a（**零新增** Python/Django HTTP 接口；全部落 Go + Vue + onlineServiceJS 契约扩展）
- **自主决策**: 层级图「发送给 AI」本期保留；一条评论最多一个 `@镜像`；ContextPack 超 256KB 截断最早评论并标 `truncated`

---

## 1. 问题 / 意图

| # | 缺口 |
|---|------|
| 1 | 工作空间无法开关「评论里 `@` 容器镜像并自动开跑」能力；默认应关闭 |
| 2 | 任务评论区无 `@镜像` mention；无法一键按项目运行机器配置拉起指定镜像 |
| 3 | 被 `@` 的容器缺少「任务详情 + 截至当前评论的完整线程」上下文与自动执行入口 |
| 4 | 容器内智能体缺少**回写任务评论**的正式接口，且需支持 **SSE 持续流式回复** |
| 5 | 产品希望**模糊 AI 评论区 vs 人类评论区**：有 `@镜像` = 发给 Agent 并执行回复；无 `@` = 朴素评论 |

### 已确认决策（用户）

| 项 | 决策 |
|----|------|
| Q1 `@` 候选 | 仅本租户 `tenant_installed_images` |
| Q2 机器策略 | 优先复用工作空间闲置机（`prefer_idle_reuse`），否则再新开 |
| Q3 回写模型 | 新建「容器 Agent 评论」类型，挂在人类评论线程下，落 Go |
| Q4 上下文 | 任务详情 + 从最早到当前这条的全部人类/AI/Agent 评论 |
| Q5 开关 | 工作空间级，默认关；关则前端不解析 `@镜像`，后端拒绝触发开机器 |
| 补充 UX | 统一评论输入；`@镜像`→Agent 路径；无 `@`→朴素评论 |

---

## 2. 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | 工作空间可开关 `container_image_at_mode_enabled`，默认 `false` | task-panel UI + PATCH workspaces 回显 |
| S2 | 开关关闭时：前端无 `@` 镜像候选；后端即使伪造 mention 也不启动机器 | 单测 + E2E |
| S3 | 开关开启：评论输入 `@` 可从租户已安装镜像中选；提交后落人类评论并带 mention 元数据 | API + UI |
| S4 | 含合法 `@镜像` 的评论 → 走 start-vm/start-vm-auto，且遵守 `prefer_idle_reuse` | Cloud 单测 / 事件断言 |
| S5 | 容器拉取/启动上下文含：任务详情 + 评论线程（含当前触发评论） | container task-detail / bootstrap 契约单测 |
| S6 | 容器可经鉴权接口流式回写「容器 Agent 评论」（SSE 增量 + 最终落库） | Go 单测 + 前端流式展示 |
| S7 | 无 `@镜像` 的提交 = 仅人类评论，不启机器、不建 Agent 评论 | 单测 |
| S8 | 无新增 Python HTTP 接口；Swagger 对新 Go 接口可见 | 路由归属 / OpenAPI |
| S9 | 价值流 / 意图文档登记本能力 | step 3 细化 |

---

## 3. 方案对比与采纳

### 3.1 开关落点

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. `workspaces.container_image_at_mode_enabled`**（taskProjectService） | 与 `task_archive_tier` 同级工作空间功能开关 | **采纳**：产品语义是工作空间功能，非云策略 |
| B. `workspace_machine_policies` | 与闲置复用绑在一起 | 拒：机器策略关心「怎么开」，本开关关心「能不能 @」 |
| C. feature-params | 过重、易与个人参数混淆 | 拒 |

### 3.2 统一评论 vs 存量 AI 评论

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. 评论区单一 composer** | 提交时后端/前端按是否含 `@镜像` 分流；会话流统一展示 | **采纳** |
| B. 保留独立「发送给 AI」评论按钮 | 与「模糊区域」冲突 | 拒（评论区） |
| C. 层级图「发送给 AI」一并移除 | 层级图是 job/层锚点指令，语义不同 | **本期保留**；不在本迭代删除 |

说明：当前 `TaskDetailCommentsPanel` 已是单一「提交评论」；`提交给 AI` 主要在 **层级图面板**。本期统一的是**评论语义**（`@`=Agent），层级图路径不动。

### 3.3 容器 Agent 评论落点

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. 扩展 taskAIComment** | 新表/新 kind=`container_agent`；复用 SSE / Kafka / internal PATCH 经验 | **采纳**：流式与容器出站已在此域 |
| B. 塞进 taskTaskService `comments` | 人类评论与长连接流式混在一库 | 拒：SSE/worker 边界不清 |
| C. 新建独立 Go 服务 | 过度设计 | 拒 |

### 3.4 编排触发点

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. taskTaskService 创建评论后发 Kafka** → taskEvents/taskAIComment 消费编排 | 解耦、可重试 | **采纳** |
| B. 前端先启机器再发评论 | 离开页面易丢；双源真相 | 拒 |
| C. 仅 Django 编排 | 违反 Go-first | 拒 |

---

## 4. 领域概念（轻量，供 `/6-ddd`）

| 概念 | Bounded Context | 说明 |
|------|-----------------|------|
| **WorkspaceAtModePolicy** | Project/Workspace | 工作空间是否允许镜像 `@` |
| **ImageMention** | Task | 评论中对 `tenant_installed_images` 的结构化引用 |
| **HumanComment** | Task | 人类朴素评论（可含 mention 元数据） |
| **ContainerAgentComment** | AI Comment | 挂在 `parent_comment_id` 下的 Agent 回复线程节点 |
| **AtMentionRun** | AI Comment / Cloud | 一次 `@` 触发的开机器→跑容器→注入上下文→执行→流式回写 |
| **CommentThreadContextPack** | Task + AI Comment | 任务详情 + 截至触发评论的全量线程快照 |

**候选领域事件**

| 事件 | 生产者 | 消费者 |
|------|--------|--------|
| `TASK_COMMENT_IMAGE_MENTIONED` | taskTaskService | taskAIComment（编排）/ taskEvents |
| `CONTAINER_AGENT_STREAM_CHUNK`（或复用 SSE，不落 Kafka） | taskAIComment | 浏览器 SSE |
| `CONTAINER_AGENT_REPLY_COMPLETED` | taskAIComment / onlineServiceJS 回调链 | taskEvents（可选审计） |

---

## 5. 详细设计

### 5.1 工作空间开关

**表**（`task-project-service` / `workspaces`）：

```sql
ALTER TABLE workspaces ADD COLUMN container_image_at_mode_enabled INTEGER NOT NULL DEFAULT 0;
```

- API：既有 `GET/PATCH /api/tenant/{t}/workspaces/{id}/` 增字段（布尔）
- UI：`WorkspaceSettingsTaskPanel.vue` 增加开关（默认关），文案建议：「开启容器镜像 @ 模式」
- 任务详情页读取工作空间字段（任务详情聚合或独立 workspace GET）驱动前端 mention 能力

### 5.2 统一评论提交流程

```text
用户提交评论
  ├─ 开关 OFF 或正文无合法 @镜像
  │    → taskTaskService 仅创建人类评论 → 结束
  └─ 开关 ON 且含 ≥1 合法 @镜像
       → 创建人类评论（content + mentions[]）
       → 创建 ContainerAgentComment（pending，parent=人类评论）
       → 发布 TASK_COMMENT_IMAGE_MENTIONED
       → 编排：解析镜像 → start-vm(-auto)+idle reuse → 等待可达
       → 向容器注入 ContextPack + 触发自动执行（当前评论文本为指令）
       → 容器经回写 API SSE/分片推送 → 浏览器统一 Feed 展示
```

**Mention 语法（前后端契约）**

- 存储：结构化 `mentions: [{ type: "installed_image", id, name }]`
- 展示：正文可保留 `@镜像显示名`；提交体同时带结构化字段，避免纯文本解析歧义
- MVP：**一条评论最多一个镜像 mention**；多个 → `400` 提示「请只 @ 一个镜像」
- 候选源：`GET` 租户已安装镜像列表（既有 installed-image API）

**评论区 Composer UX（2026-07-18）**

- 开关开启时使用 **contenteditable 富文本输入**（非独立 `<select>`）：输入 `@` 弹出已安装镜像下拉；继续键入按名称过滤；`Enter`/`Tab`/点击选中，或键入完整镜像名后以**空格**确认。
- 选中后正文插入高亮 chip（`@显示名`），并以空格与后续指令文本分隔；结构化 mention 仍经 `pendingImageMention` → 提交体 `mentions[]`。
- 开关关闭时回退普通 `<textarea>`，不解析 `@`。

**后端硬闸**

即使前端绕过，`taskTaskService` 创建评论时：

1. 读 workspace 开关（调 taskProjectService internal 或缓存）
2. 开关关且带 mentions → `400` / 忽略 mentions 且**不**发事件（推荐硬拒 `403/400`）
3. mention id 不在本租户 `tenant_installed_images` → `400`

### 5.3 机器与容器

复用既有链路，**不新造开机器 API**：

1. 按任务所属**项目** `server_run_template` 组装 start-vm / start-vm-auto（同 `buildAutoRunStartVmRequest`）
2. `container_image_id` = 被 `@` 的已安装镜像 id（可覆盖任务默认镜像）
3. `taskCloudService` 入口已含 `applyStartVmPolicyGateAndReuse` → **优先闲置复用**
4. 镜像 URL 仍由 Cloud 解析 `tenant_installed_images`

幂等建议：同一 `parent_comment_id` 只允许一个进行中的 `AtMentionRun`；重复提交返回已有 run。

### 5.4 ContextPack（注入容器）

容器侧（`onlineServiceJS` task-detail / bootstrap 扩展，**非新 Python 服务**）增加例如：

```json
{
  "at_mention_run": {
    "run_id": "...",
    "parent_comment_id": "...",
    "agent_comment_id": "...",
    "trigger_comment": { "id": "...", "content": "...", "created_by": "..." },
    "installed_image": { "id": "...", "name": "..." }
  },
  "task": { /* 既有 task-detail */ },
  "comment_thread": [
    { "kind": "human|ai|container_agent", "id": "...", "content": "...", "assistant_response": "...", "created_at": "..." }
  ]
}
```

- `comment_thread`：按时间从最早到**含当前触发评论**
- 含历史人类评论、历史 AI 评论、历史容器 Agent 评论（若有）
- 自动执行：bootstrap/就绪后创建 job，`command` = 触发评论正文（去掉 mention 标记后的指令文本，或保留全文——实现时定一种并写进契约）；`command_kind` 与现有 Trae/agent 一致

长度：MVP **全量注入**；若超限（实现阶段定阈值，如 256KB），截断最早评论并在 pack 中标注 `truncated: true`（预期问题前置：须单测覆盖）。

### 5.5 容器 Agent 评论 + SSE 回写

#### 数据（taskAIComment）

```sql
CREATE TABLE IF NOT EXISTS container_agent_comments (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  task_id TEXT NOT NULL,
  parent_comment_id TEXT NOT NULL,      -- 人类评论 id（taskTaskService）
  installed_image_id TEXT NOT NULL,
  run_status TEXT NOT NULL,             -- pending|starting|running|streaming|completed|failed
  content TEXT,                         -- 可选：系统摘要「已 @ xxx 启动」
  assistant_response TEXT,              -- 最终完整回复
  created_at DATETIME,
  updated_at DATETIME
);
```

会话 Feed：前端合并 `human comments` + `ai_task_comments` + `container_agent_comments`（按时间 / parent 挂载），**视觉上同一时间线**；Agent 节点可显示镜像名徽章，但不拆独立输入区。

#### 容器 → 平台回写 API（Go）

鉴权：容器持有的 **container access token**（既有 taskCredentialService / taskAgentSupport 入站模式）。

建议落在 **taskAIComment**（公网经网关）或经 **taskAgentSupport** 云入站转发到 taskAIComment internal：

| 方法 | 路径（草案） | 说明 |
|------|--------------|------|
| `POST` | `/api/tenant/{t}/workspace/{ws}/task/{task}/container-agent-comments/{id}/stream` | **SSE 或 chunked**：持续追加回复；`Content-Type: text/event-stream`（容器作客户端推送可用长 POST + chunk，平台再 fan-out SSE 给浏览器） |
| `POST` | `.../container-agent-comments/{id}/complete` | 标记完成、落最终 `assistant_response` |
| `POST` | `.../container-agent-comments/{id}/fail` | 失败原因 |

**浏览器侧 SSE**：复用任务级 `server-startup-status-sse`，新增 status 如 `container_agent_stream`（phase=`chunk|done|error`），与现有 `ai_instruct_stream` 并列；前端写入同一 stream buffer 或按 `agent_comment_id` 分槽。

> 设计要点：容器「持续回复」≠ 浏览器「持续订阅」。平台作为中枢——容器推分片 → 平台落缓冲/落库 → SSE 推给打开任务详情的用户。

### 5.6 前端

| 点 | 行为 |
|----|------|
| task-panel 设置 | Toggle 开关，PATCH workspace |
| TaskDetail composer | 开关 ON：`@` 弹出已安装镜像；选中插入 mention chip + 结构化 payload |
| 提交按钮文案 | 可选：检测到 mention 时显示「提交并运行」；无 mention 仍「提交评论」 |
| Feed | 统一时间线；streaming 中显示打字/流式块挂在对应 parent 下 |
| 开关 OFF | 无 mention UI；提交仅人类评论 |

### 5.7 权限

- 开关：工作空间管理员（与改 `task_archive_tier` 同级）
- `@` 触发：具备任务评论权限者
- 容器回写：仅有效 container token，且 token 绑定的 task/workspace 与 comment 一致

### 5.8 Swagger

所有新增/变更 Go 路径须进对应服务 OpenAPI（taskProjectService / taskTaskService / taskAIComment），含 4xx 说明。

---

## 6. 接口清单（Go）

| # | 服务 | 变更 | 公网/internal |
|---|------|------|---------------|
| 1 | taskProjectService | workspace 字段 R/W | 公网（既有 path） |
| 2 | taskTaskService | POST comments 支持 `mentions`；开关校验；发 Kafka | 公网扩展 |
| 3 | taskAIComment | CRUD/查询 container_agent_comments；stream/complete；SSE 发布 | 公网 + internal |
| 4 | taskCloudService | **无新 path**；编排调用既有 start-vm(-auto) | — |
| 5 | taskEvents | 消费 mention 事件（若编排放此）或仅审计 | internal 消费 |
| 6 | taskAgentSupport | 可选：转发容器入站 stream 到 taskAIComment | 扩展既有入站 |
| 7 | onlineServiceJS | ContextPack + 自动 job | 容器内 API 契约 |

**不触发 Python 接口额外审批门**（无新 Django/Flask endpoint）。

---

## 7. 价值流影响（输入给 `/3-value-stream`）

| Stream | 影响 |
|--------|------|
| `task-management` / `task-comments-api` | 扩展 mentions + 分流 |
| `task-detail-runtime-relay` / `container-runtime-context` | ContextPack、Agent 回写 |
| `cloud-integration` / `workspace-machine-idle-policy` | `@` 触发路径必须走 idle reuse |
| `project-workspace` | workspace 新字段 |
| **建议新 step** | `container-image-at-mention`（planned→active） |

交叉依赖：评论域 → 云开机器 → 容器运行时 → Agent 评论 SSE。

字段示例（三元）：

- `task-project-service.workspaces.container_image_at_mode_enabled`
- `task-task-service.comments.mentions_json`（或旁表）
- `task-ai-comment.container_agent_comments.run_status`
- `task-ai-comment.container_agent_comments.assistant_response`

---

## 8. 🏛️ 架构变更影响（批准后写入文件）

批准后将创建 **v26** `application-integration` 三类伴生文件（`.puml` + `.archimate` + `.mermaid.md`），**不修改** v25 current。

| 标记 | 内容 |
|------|------|
| 🟢 NEW | `ContainerAgentComment` 数据对象；`TASK_COMMENT_IMAGE_MENTIONED` 流；容器回写 stream 接口 |
| 🟡 MODIFIED | Vue TaskDetail 评论区；taskTaskService comments；taskAIComment；taskProjectService workspaces；onlineServiceJS ContextPack |
| 🟡 MODIFIED | 编排复用 taskCloudService start-vm + idle reuse（关系增强，无新服务） |
| 🔴 无 | 不废弃层级图「发送给 AI」 |

### .archimate 架构变迁要点（批准后落实）

| 元素 | 内容 |
|------|------|
| Plateau v25 | current — 自动 SG 白名单 |
| Plateau v26 | target — 镜像 `@` 模式 + 统一评论 Agent 回写 |
| Gap | 无 `@` 镜像触发；无容器 Agent 评论 SSE 回写；评论区与 Agent 路径未统一 |
| WorkPackage | 开关 + mention 评论 + 开机器编排 + ContextPack + 容器 stream API + Feed |

**并行 target 提示**：v22/v23/v24 仍为 target 积压；本设计基于 **v25 current**，版本号取 **26**。

---

## 9. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 冷启动慢，用户以为卡住 | Feed 显示 starting/running 状态；SSE 阶段事件 |
| 全量线程过大 | 阈值 + truncated 标记；后续可摘要 |
| 与任务默认镜像不一致 | `@` 镜像优先；UI 标明将运行的镜像 |
| 与层级图 AI 双通道混淆 | 文档/文案区分：「评论 @镜像」vs「层级指令」 |
| 开关关闭后历史 Agent 评论 | 只读保留；不可新触发 |

---

## 10. 非目标（本期不做）

- `@用户` / `@项目` 等其他 mention
- 一条评论 `@` 多个镜像并行开多机
- 删除层级图「发送给 AI」
- 迁移/删除历史 `ai_task_comments` 表
- 新建独立微服务

---

## 11. 审批检查清单

- [ ] 总体设计批准（本文件）
- [ ] 确认层级图「发送给 AI」本期保留
- [ ] 批准后写入 v26 架构三类文件 + VERSION_HISTORY
- [ ] 下一步：worktree 或价值流
