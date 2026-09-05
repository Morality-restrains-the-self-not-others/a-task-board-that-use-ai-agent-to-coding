# 创建任务：镜像/技能 mention 存储 ID 化 + 名↔ID 映射持久化 — 设计文档

- **日期**: 2026-08-24
- **作者**: claude
- **状态**: approved（2026-08-24，用户确认 D1=B 服务端派生、D2=B ID+名字快照）
- **关联迭代**: taskFE 创建任务弹窗技能 chips（已完成）；本设计为其后端存储契约延伸

## 1. 背景与现状

### 1.1 用户需求（原始表述）

> 前端显示的时候显示 `$aaa` 和 `/bbb`；后端存储的时候，应该把 `aaa` 和 `bbb` 改为对应的 ID。镜像在保存的时候，就需要把「镜像名与 ID 的映射」以及「技能名与 ID 的映射」进行存储。

拆解为三条可执行标准：

1. **显示层不变**：描述文本仍显示可读的 `$镜像名 /技能名`（如 `$trae-agent /general-coding`）
2. **存储层 ID 化**：任务保存时，镜像与技能均以 **ID** 落库（镜像 ID 已有；**技能 ID 目前不存在**，须引入）
3. **映射持久化**：保存时把「镜像名↔ID」「技能名↔ID」映射持久化，使后端不依赖前端传名反查，镜像目录变化后任务仍可正确解析

### 1.2 现状链路（已核对源码）

| 环节 | 现状 | 文件 |
|------|------|------|
| 前端显示 | 描述 textarea 写入 `$name /skill` mention 文本；chips 提示 `$镜像名 /技能名` | taskFE `TaskDescriptionSkillField.vue` |
| 前端草稿绑定 | `editingTask.container_image = {id, mentionText, skill: <技能**名**>}` | taskFE `CreateTaskBasicFields.vue` |
| 前端提交 | `applyImageMentionToTaskDescription` 把 mention 文本合并进描述自由段；payload 带 `container_image_id` | taskFE `CreateTaskModal.vue:363` |
| 后端存储 | `task_tasks.installed_image_id` = 镜像 ID；`description` 原文含 `$name /skill` 文本 | taskTaskService `task_store.go` / `create_task.go:141` |
| 技能数据 | **无技能 ID 概念** — `imageSkills.yaml` 无 id；`cloud_tenant_installed_images.image_skills_json` 仅 `{name, description, is_default}` | taskCloudService `parse_image_skills.go` / `extract_image_skills.go`；迁移 `dataMigrate/taskCloudService/035_image_skills.sql` |
| 读取回填 | `hydrateTaskContainerImage` 返回 `container_image {id, name, version}`（无技能）；技能靠前端从描述文本 + 目录 `image_skills` 反解显示 | taskTaskService `created_by_enrich.go:138` |
| 服务端校验 | auto_run 前置校验仅用 `container_image_id` | taskTaskService `create_task.go` |

**关键事实**：
- 镜像名↔ID 映射天然存在于 `cloud_tenant_installed_images`（id UUID + name），但**未在任务上持久化快照**；镜像卸载后任务 `container_image_removed=true`，名字丢失
- 技能名 `^[a-z0-9][a-z0-9-]{0,62}$` 本身是合法 slug，但**没有独立 ID 字段**，无法支撑「改名安全」与「版本演进」
- 评论区 mention 仅支持 `$镜像`（无技能），本次不涉及

## 2. 目标设计

### 2.1 总览

```
前端显示: $trae-agent /general-coding        （可读文本，不变）
         │ 提交
         ▼
API 请求: { container_image_id: <镜像ID>, container_image_skill_id: <技能ID>,
            description: "... $trae-agent /general-coding ..." }
         │
         ▼
后端存储: task_tasks.installed_image_id = <镜像ID>
          task_tasks.image_skill_id      = <技能ID>          （新增列）
          task_tasks.container_image_snapshot = {"image_id","image_name","skill_id","skill_name"}  （新增列，映射快照）
         │
         ▼
API 响应: container_image = { id, name, version, skill: { id, name } }  （反解回填）
```

### 2.2 设计决策

#### D1 — 技能 ID 生成方式（**待用户确认，见审批问题 Q1**）

| 方案 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **A. imageSkills.yaml 显式 id** | 镜像作者在 yaml 中声明 `id` | 镜像作者可控、可稳定演进 | 改所有镜像发布流程；存量无 id 需回填 |
| **B. 服务端提取时派生**（推荐） | taskCloudService 抽取 `imageSkills.yaml` 时生成稳定 ID（如 `sk_<sha1(name+镜像external_id)>[:12]`），写入 `image_skills_json` | 镜像作者零改动；单点实现；存量一次回填 | ID 与名字同构（改名即变 ID，绑定会失效）——但技能改名本就破坏绑定，行为可预期 |
| **C. 技能名即 ID** | name 已是合法 slug，直接以 name 为 ID | 零新增字段 | 无「真 ID」语义，映射退化恒等；无法支撑未来技能版本化 |

> 方案 B 与 C 在存储层可统一：`image_skills_json` 每技能附 `id` 字段（B 派生 / C 恒等 name），任务引用 `skill_id`；前端与后端均以 `id` 为准，name 仅作显示。**推荐 B**，理由：镜像作者无感、改动收敛在 taskCloudService 一处、为技能改名/版本演进留出独立 ID 空间。

#### D2 — 映射存储位置（**待用户确认，见审批问题 Q2**）

| 方案 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **A. 纯 ID 引用** | 任务只存 `installed_image_id` + `image_skill_id`；名字读取时查目录反解 | 存储最小；目录是唯一真源 | 镜像卸载/改名后名字丢失（现已有 `container_image_removed` 场景） |
| **B. ID + 名字快照**（推荐） | 任务新增 `container_image_snapshot` JSON（`{image_id, image_name, skill_id, skill_name}`），描述文本仍存 `$name /skill` | 满足用户「保存时把名↔ID 映射进行存储」；镜像目录变化不影响任务显示与解析；编辑回填零查库 | 冗余存储；镜像改名后任务显示旧名（快照语义，需注明） |

> 用户原话「镜像在保存的时候，就需要把镜像名与 ID 的映射以及技能名与 ID 的映射进行存储」明确指向**保存时落映射** → 方案 B。快照为展示/校验用途，目录仍是权威真源（任务执行用 `installed_image_id` 实时解析）。

#### D3 — 描述正文形态（自主决策，不询问）

描述自由段**保留** `$name /skill` 可读文本（不改为 ID 形态），理由：
- 描述会进入任务执行上下文（LLM 提示词），可读性优先
- 现状 round-trip 契约（`applyImageMentionToDescription` / `stripImageMentionFromDescription`）以名字为显示锚点，改动面最小
- 结构化 ID 由新增列承载，描述文本与结构化绑定通过「名↔ID 映射」互校验（防伪造/防错配）

#### D4 — API 契约变更

**taskTaskService 创建/更新任务**：
- 请求体新增可选字段 `container_image_skill_id`（string）
- 服务端校验（fail-closed）：
  1. `container_image_id` 存在且属于本租户（沿用现有 lookupInstalledImageFn）
  2. `container_image_skill_id` 存在 → 必须在 `image_skills_json` 的技能列表中，否则 400 `code: skill_not_in_image`
  3. `container_image_skill_id` 缺失但描述含 `/技能` → 400 `code: skill_id_required`（前端必带；防御性）
  4. 快照写入：`image_id/image_name` 取自目录，`skill_id/skill_name` 取自 image_skills_json
- 响应 `container_image` 对象扩展 `skill: {id, name}`（缺失时省略，兼容存量）

**taskCloudService**：
- `image_skills_json` 序列化结构每技能附 `id`（D1 方案落地处）
- 存量回填迁移（`dataMigrate/taskCloudService/041_image_skill_ids.sql`）：遍历非空 `image_skills_json`，为缺失 id 的技能按 D1 规则生成（seed = external_image_id 空回退 id；与 Go `deriveImageSkillID` 规则一致，有跨侧一致性测试）

**taskFE**：
- `normalizeImageSkillList` 透传 `id`（前端忽略未知字段，天然向后兼容）
- 提交 payload 携带 `container_image_skill_id`（由 `container_image.skill` 名字 → 目录反查 id，或直接存 skill_id）
- 编辑回填：`container_image.skill_id` → 目录反解显示；快照存在时优先快照

### 2.3 兼容与迁移

| 场景 | 处理 |
|------|------|
| 存量任务（installed_image_id 有、image_skill_id 空） | 读取时 skill 从描述文本解析（现状行为）；快照缺失不报错 |
| 存量镜像（image_skills_json 无 id） | 迁移回填；回填后新任务保存即可获得 skill_id |
| 老前端（不带 container_image_skill_id） | 服务端容错：描述含 `/技能` 且可反解时服务端按名反解 skill_id（尽力而为，非 fail-closed）；描述无技能 → skill 列置空 |
| 技能改名 / 镜像卸载 | 快照保留旧名显示（标注「已卸载/已变更」由既有 `container_image_removed` 机制承担） |

## 3. 🕸️ Code Review Graph 分析

- `graph.db` 存在；`createTaskOnce` 节点未命中（该符号定义于 `create_task.go` 动态分派，CRG 未收录）。以手工源码核对为准（§1.2 已列全部相关文件与行号）。
- 影响面：taskFE 3 文件（TaskDescriptionSkillField / CreateTaskBasicFields / CreateTaskModal + utils 2 个）+ taskTaskService 3 文件（create_task / task_handlers_update / created_by_enrich + task_store）+ taskCloudService 2 文件（parse_image_skills / extract_image_skills / installed_image_handlers / installed_image_store）+ 迁移 2 个（taskCloudService 041 / taskTaskService 016）。

## 4. Domain Concept Inventory

- **Bounded Contexts**: 任务协作（taskTaskService）、云镜像目录（taskCloudService）、前端编排（taskFE）
- **Key Entities**:
  - `InstalledImage`（cloud_tenant_installed_images）— 已有 id/name；技能列表 `image_skills_json`
  - `ImageSkill`（本次引入 ID）— 镜像内技能，`{id, name, description, is_default}`
  - `Task`（task_tasks）— 新增 `image_skill_id` + `container_image_snapshot`
- **Candidate Aggregates**: Task（镜像绑定为任务值对象）
- **Domain Events**: 无新增（见下）

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| 创建/更新任务并绑定镜像技能 | — | — | — | **纯存储契约变更**：技能 ID 生成与任务快照均为既有「创建任务」意图的落库形态变化，不改变系统事实语义、无跨边界副作用（事件总线消费方不依赖 skill 结构）。若未来需要「技能绑定审计/变更流」，另行引入 `TaskImageSkillBound` 事件 |

## 6. Value Stream Impact

- 受影响流：`任务协作` 域「创建任务」相关 step（conf/value-stream.yaml L622 起）— 字段影响 `taskTaskService.task_tasks.image_skill_id`（新增）、`taskCloudService.cloud_tenant_installed_images.image_skills_json`（结构扩展）
- 测试影响：taskTaskService create/update 用例（新增 skill_id 校验）、taskCloudService 技能解析用例、taskFE 提交契约用例
- 无新价值流；步骤状态不变

## 7. 🏛️ 架构变更影响

- **迭代版本**: v107 🎯 target
- **迭代名称**: 镜像/技能 mention 存储 ID 化 + 名↔ID 映射持久化
- **作者**: claude
- **涉及视图**: `enterprise-landscape`、`application-integration`（数据流/存储契约变更）
- **变更明细**:
  - 🟡 [MODIFIED] taskTaskService — task_tasks 新增 image_skill_id + container_image_snapshot；创建/更新 API 新增 container_image_skill_id；响应 container_image.skill
  - 🟡 [MODIFIED] taskCloudService — image_skills_json 技能附 id；迁移回填
  - 🟡 [MODIFIED] taskFE — 提交携带 skill_id；回填显示反解
  - 🔴 无废弃
- 伴生文件（每个视图四类，均已生成并通过 Archi `--loadModel` 验证）:
  - `v107-application-integration-20260824-1515-claude.puml` + `.diff.archimate`（增量：v106→v107 变迁链 Plateau/Gap/WP + sourceConnection）+ `.full.archimate`（全量：gw/fe/tts/cloud/django + 2 DataObject 拓扑）+ `.mermaid.md`
  - `v107-enterprise-landscape-20260824-1515-claude.puml` + `.diff.archimate`（增量：v106→v107 变迁链 + 目标拓扑）+ `.full.archimate`（全量：dev/fe/tts/cloud + MySQL 拓扑）+ `.mermaid.md`
- 架构版本历史: `docs/architecture/VERSION_HISTORY.md` 已登记 v107 🎯 target 条目（基于 v106）

## 8. 🐍 Python 新增接口清单与 Go 替代评估

**未触发** — 本设计仅在 Go 服务（taskTaskService / taskCloudService）与前端（taskFE）落地，无任何 Python 服务新增接口。不适用额外审批门。

## 9. 风险与注意事项

- **技能 ID 生成规则的稳定性**：方案 B 派生 ID 须确定性（同镜像同技能每次生成一致），digest 变化触发重抽时 ID 不变
- **描述文本与结构化绑定一致性**：提交时服务端校验「描述中的 `$name /skill` 文本 ↔ container_image_id/skill_id」匹配，防前端伪造错配
- **auto_run 链路**：`validateAutoRunPrerequisites` 只依赖镜像 ID，不受技能列影响（无回归）
- **快照语义**：快照是展示/校验用，任务执行以目录实时解析为准；文档须注明防误解
