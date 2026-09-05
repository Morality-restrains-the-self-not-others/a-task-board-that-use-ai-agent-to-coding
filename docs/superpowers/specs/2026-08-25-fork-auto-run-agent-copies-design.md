# 创建设计：Fork 自动运行按智能体（模型）复制

**日期**: 2026-08-25  
**状态**: approved（2026-08-25）  
**批准记录**: 术语=智能体资源配置 + 可切换模型名；仅派生始终 1 份；单选模式 +「确认派生」  
**范围**: 任务详情页 Fork 确认模态 `#fork-auto-run-confirm-modal` 中「副本数量」章节  
**页面**: `/tenant/{tid}/workspace/{wid}/task-detail/{taskId}/`（标题「云端开发」为当前任务名）  
**既有意图**: `docs/intents/frontend/task_detail_fork_auto_run_confirm`  
**既有设计**: `docs/superpowers/specs/2026-08-23-fork-copy-count-design.md`

## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | Nodes 114 / Edges 1012 / Files 18；branch `main`；updated 2026-08-25T12:49:05。未索引本弹窗。 |
| 关键发现 | `search ForkAutoRunConfirmModal` / `forkTask` 为 0。爆炸半径改由源码调用链给出。 |
| 决策影响 | 自动运行异构副本不能复用同构 `fork_count`；模型切换须接到既有 `agent_models` → `TASK_AGENT_MODEL` 覆盖链。 |
| skip 理由 | 不 skip；`CRG unavailable for this symbol: graph indexed 18 files only`。 |

静态调用链：

`TaskDetailPageHeader.openForkConfirm` → `ForkAutoRunConfirmModal` → `forkTask` → `POST /api/tasks/todos/...` →（`auto_run`）`ensureAutoRunAtComment` → `notifyContainerAgentPending` → 容器首 job。

层级图「选择模型」已走同一覆盖：`agent_models: [{provider, model}]` → `taskAIComment/container_stream.go` 写入 `TASK_AGENT_MODEL` / `TASK_AGENT_MODEL_PROVIDER`。

## 当前架构理解

全仓 current 为 **v104**。本需求不新增服务/路由/表所有权，**不写 ArchiMate 新版本**（与 2026-08-23 copy-count 同一判定：既有 POST/内部通知增加可选字段）。

- **应用层**: taskFE 弹窗；taskTaskService 创建+auto_run；taskCloudService 解析 feature-params；taskAIComment 执行首 job 时覆盖模型 env。
- **数据**: `task_tasks.feature_params_source` / `personal_feature_params_config_id`；配置内 `agent_model` + `agent_model_provider` + `providers[].supported_models`。

📋 最近版本：v104 ✅ current；v105–v107 🎯 target 与本需求正交。

## 用户确认的术语

| 用户用语 | 映射 | 现网对应 |
|----------|------|----------|
| **智能体资源** | 智能体资源配置 | `feature_params_source`：公司默认 / 工作空间默认 / 个人配置（个人须再选 `personal_feature_params_config_id`） |
| **智能体** | 该配置里可切换的**模型名称** | 与任务详情层级图「模型（单选）」同一清单：所选 `agent_model_provider` 的 `supported_models`（缺省项为配置的 `agent_model`） |
| 已安装镜像 / 技能 | **不在本弹窗改** | Fork 继续拷贝源任务 `container_image_id` / `image_skill_id` |

## 问题与目标

自动运行时，「副本数量」整节改为：先选智能体资源，再多选智能体（模型名）；选几个就复制几份，每份绑定该模型并自动运行。仅派生时这一配置不影响结果。

## 方案对比

| 方案 | 说明 | 取舍 |
|------|------|------|
| **A. 每份独立 POST**，`feature_params_source` 相同，`agent_models` 不同 | 不改路由；每份 `Idempotency-Key=${batch}:${i}`；auto_run 把该份模型写入首 job env | **采用** |
| B. 扩展 `fork_count` 批接口带 `fork_variants[]` | 单 RTT | 本迭代拒绝：现网批创建复用同一 body，改异构会搅配额部分失败语义 |
| C. 只改 feature_params_source、不传模型 | 同配置默认 `agent_model` 全相同，无法「不同智能体」 | 拒绝 |
| D. 为每个模型新建一份 feature-params 配置 | 污染设置页 | 拒绝 |

**仅派生**: 始终 1 份，无数字框。智能体选择器隐藏且不得写入 payload。服务端 `fork_count` 批接口保留给其他调用方，本弹窗不再使用。

**🐍 Python**: `not_applicable`。

## 为何必须改 auto_run 接线（不只改 UI）

容器启动时 `TASK_AGENT_MODEL` 来自 feature-params **默认** `agent_model`。层级图切换模型靠评论 `agent_models` 覆盖首 job env。今日 `ensureAutoRunAtComment` / `notifyContainerAgentPending` **不传** `agent_models`，因此只改副本数字无法让各副本跑不同模型。

本设计把层级图已有契约接到 auto_run：每份创建请求带一个模型，写入 pending agent 的 job context，首 job 覆盖 env。

不新增 MQ 事件；每份仍 `TASK_CREATED`。无新 HTTP 路径（扩展既有 todos POST 与 internal pending-agent 可选字段）。

## 交互（推荐）

打开模态复位。派生方式单选，副本章节随模式切换：

```
确认派生任务
…

派生方式
  ○ 不自动运行，仅派生
  ○ 自动运行并派生

── 仅派生 ──
  （无数量框；确认即创建 1 份，auto_run=false）

── 自动运行 ──
  Git 身份 / OAuth（既有门禁）
  智能体资源  [公司默认 | 工作空间默认 | 个人配置]
              （个人：再选具体个人配置）
  智能体      [该资源配置下可切换模型名，多选 checkbox]
  提示：将创建 N 个副本，每个使用不同模型并尝试启动云资源。

[取消]  [确认派生]
```

### 自动运行规则

1. 智能体资源选择器复用 `ServerConfigFeatureParamsBlock` 的 source 语义（公司/工作空间/个人 + 个人配置）。打开弹窗拉可用性；禁止轮询。
2. 默认选中源任务的 `feature_params_source`（及个人配置 id）。
3. 选中资源后，按与 `createLayerGraphModelOptionsState` 相同规则拉模型清单（`agent_model_provider` 的 `supported_models`，默认模型可置顶）。
4. 默认勾选配置的 `agent_model`（若在清单中）；用户可加选。至少 1 个，最多 99。切换资源时清空并预勾新配置默认模型。
5. N = 勾选模型数；不再出现数字 input。
6. 未选资源、个人未选具体配置、0 个模型、Git/OAuth 未过 → 确认 disabled。
7. 清单为空：提示先到智能体资源配置页补 `agent_model_provider` 与支持模型；不可确认自动运行。
8. 每份 POST：
   - `auto_run: true`
   - `feature_params_source`（个人则含 `personal_feature_params_config_id`）
   - `agent_models: [{ provider, model }]`（一项；provider = 该配置的 `agent_model_provider`）
   - 镜像/技能/描述仍从源任务拷贝
   - `Idempotency-Key=${batch}:${i}`
9. **禁止**自动运行路径使用 `fork_count`（同构批会丢掉每份模型）。
10. N=1 打开该任务；N>1 只打开第一份。部分失败语义同现网。
11. 确认点击：既有 `createClickGuard` + 同意图同一 batch 键。

### 仅派生

不展示资源/模型/数量框；不写 `agent_models`；`copyCount` 恒为 1。不要求 Git/OAuth。用户已确认去掉仅派生的 1–99 数字框。

### 底部按钮

已批准：单选模式 +「取消」「确认派生」。不再保留「自动运行并派生」「不自动运行，仅派生」双按钮。

## 服务端落点

| 位置 | 变更 |
|------|------|
| `handleCreateTask` / `createTaskOnce` | 读取可选 `agent_models`（auto_run 时校验：须为 1 项，model 非空，provider 非空） |
| `autoRunTriggerParams` | 增加 `AgentModels` |
| `ensureAutoRunAtComment` → `notifyContainerAgentPending` | 把 `agent_models` 写入 pending agent `context_pack` / job context |
| taskAIComment 容器 agent 启动首 job | 从 context_pack 读 `agent_models`，走已有 `traeJobsRequestBody` env 覆盖 |
| 单测 | 创建带 agent_models 的 auto_run fork；仅派生不带；非法列表 400 |

不新增表。模型绑定在本次自动运行评论/agent 上（与层级图「这次跑哪个模型」同粒度）。任务仍绑定所选 feature_params_source，供详情页以后再切模型。

若后续要把「该副本的默认模型」持久化到任务行，另开增量（新列或 parameters）；本迭代不改 schema。

## confirm payload（前端）

```js
// 仅派生
{ autoRun: false, copyCount: 1 }

// 自动运行
{
  autoRun: true,
  copyCount: agents.length,
  featureParamsSource: 'company' | 'workspace' | 'personal',
  personalFeatureParamsConfigId: '', // personal 时必填
  agentModelProvider: 'openai',
  agents: [{ model: 'gpt-4.1' }, { model: 'gpt-4.1-mini' }]
}
```

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| Fork 仅派生 1 份 | `TASK_CREATED` | `createTaskOnce` | 看板 SSE、配额 | 不新增事件类型 |
| Fork 自动运行按模型复制 | `TASK_CREATED` × N | `createTaskOnce`（每份不同 `agent_models`） | 看板 SSE、配额、auto_run 启服与模型覆盖 | 不新增 `TASKS_FORKED_WITH_MODELS` |

## Domain Concept Inventory

- **Bounded Contexts**: 任务、智能体资源配置（cloud feature-params）、自动运行评论（AIComment）
- **Key Entities**: Task、FeatureParamsConfig、AgentModelName（配置内值对象）、AutoRunComment
- **Aggregates**: Task 创建边界；每份一次 `createTaskOnce` + 一次 auto_run 评论
- **Events**: `TASK_CREATED` only

## 价值流影响

触及 `todo-fork-from`、`create-task-auto-run-backend-start`、`create-task-auto-run-git-identity`。不新增独立价值流。字段：

- `task-task-service.task_tasks.feature_params_source`
- `task-task-service.task_tasks.personal_feature_params_config_id`
- `task-task-service.task_comments` / AIComment job context `agent_models`（JSON 路径记入 description）

## 角色权限

不新增权限点。列 feature-params 与创建任务相同。模型名必须属于所选配置的 `supported_models`（或等于默认 `agent_model`）。禁止写入未授权配置 id。

## NFR 预告

- 路径已带 `tenant_id` + `workspace_id`。
- 写路径（配额/启服）幂等 ≥ L3；键 `${batch}:${i}` 与「第 i 个所选模型」同粒度。
- 前端确认：同步门闩 + Idempotency-Key。
- 禁止轮询刷新配置。

## 🏛️ 架构变更影响

不升版。无 `v108-*` puml / `.diff.archimate` / `.full.archimate`。current 仍为 v104。  
🟡 既有 Rel_Flow 可选字段：todos POST `agent_models`；internal pending-agent `context_pack.agent_models`。

## 测试要点

| # | 场景 | 期望 |
|---|------|------|
| T3 | 仅派生确认 | 1 次 POST，`auto_run=false`，无 `agent_models`，无 `fork_count` |
| T10 | 仅派生不再提供数量框 | 不出现 `fork-copy-count-input`；1 次 POST 无 `fork_count` |
| T14 | 自动运行未选模型 | 确认 disabled |
| T15 | 公司配置勾选 gpt-4.1 与 gpt-4.1-mini | 2 次 POST，`agent_models[0].model` 不同，不用 `fork_count` |
| T16 | 仅派生即使勾过模型 | 1 份同构，不按模型分叉 |
| T17 | 切换智能体资源 | 已选模型清空并预勾新默认 |
| T18 | auto_run 创建后 pending agent 带所选 model | 单测：notify/job context 含该 `agent_models` |

## 批准结论

- 2026-08-25 用户批准。
- 仅派生始终 1 份（去掉数字框）；自动运行 N = 所选模型数。
- 交互：派生方式单选 + 底部「取消」「确认派生」。
