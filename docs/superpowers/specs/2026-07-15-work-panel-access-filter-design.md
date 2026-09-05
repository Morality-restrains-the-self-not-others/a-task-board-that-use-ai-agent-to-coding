# 设计：工作面板按可访问人或小组过滤

- 日期：2026-07-15
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 相关 URL：`https://www.daydaymoney.com/tenant/{tenantId}/work-panel/`
- 架构版本：**无需新版本**（纯前端客户端过滤 + 复用既有 API；无新服务/表/公网接口）
- `python_api_approval`: scoped-down（**零新增 Python HTTP 接口**）

## 1. 问题

工作面板（`/tenant/{id}/work-panel/`）现有过滤围绕交付物路径、机器运行态、搜索/优先级，**无法**按「可访问当前工作空间的人或小组」缩小看板任务，协作场景下难以只看某人/某组相关任务。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 过滤选项仅来自当前工作空间的访问主体 | 人：workspace access 用户；小组：workspace access 小组 |
| S1b | 「人」选项展示公司成员名称（非 user_id 前缀） | `member_name` → profile username → id 前缀；见 intent `014` |
| S2 | 按「人」过滤：任务 `owner` 或任一 `assignees` 命中该人 | 看板仅保留命中任务；清除后恢复 |
| S3 | 按「小组」过滤：任务参与者属于该组成员 | 展开组员后按 company_member_id 集合匹配 |
| S4 | UI 可发现、可清除 | Header 控件 + chip；`data-alias` 可定位 |
| S5 | 与机器运行态过滤可叠加（AND） | 两过滤同时生效 |
| S6 | 无新公网 API；意图/测试齐全 | intent `009` + 单元测 + Playwright |

**范围外（本迭代）**：

- 持久化到 `work-panel-filters`（与机器运行态一致，会话内有效；切工作空间清除）
- 服务端 todos 查询参数扩展（`assignee` 半成品不扩展为小组）
- 按权限角色（viewer/edit/admin）过滤
- 旧版模态 `filterOptions.assignee` UI 补齐（可选后续合并）

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | 客户端过滤：Header 人/小组选择器；数据来自 `workspace-permissions` + 组员 `groups/{id}/members/`；匹配 `owner`/`assignees` | 与 005 机器过滤同型；零新 API；实现快 |
| B | 扩展 todos 列表 API（Go taskTaskService）支持 `assignee`/`group_id` | 需新查询语义与跨服务解组员；本需求纯视图过滤不值得 |
| C | 仅人过滤、不做小组 | 不满足用户「人或小组」 |
| D | 持久化进 work-panel-filters payload | 用户未要求；可后续 version 字段扩展 |

## 4. 领域概念

| 概念 | 说明 |
|------|------|
| **WorkspaceAccessSubject** | 可访问当前工作空间的主体：人（含 `company_member_id`）或小组 |
| **AccessFilter** | 值对象：`{ kind: 'person' \| 'group' \| null, id, label, memberIds: string[] }` |
| **TaskParticipant** | 任务参与者 ID 集合：`owner` ∪ `assignees`（均为 **company_member_id** 字符串） |

### 匹配规则

1. `kind === null` → 不过滤。
2. `kind === 'person'` → `memberIds = [company_member_id]`；任务参与者与 `memberIds` 有交集则保留。
3. `kind === 'group'` → 拉取组成员 `user_id`，经协作人池映射为 `company_member_id` 集合；同上交集匹配。
4. 无 `company_member_id` 的 access 用户行：跳过或映射失败时不进入可选列表（避免误匹配）。
5. 空组：过滤结果为空列表（合法）。

### ID 边界

- 任务侧：`owner` / `assignees` = CompanyMember 主键（与 CreateTaskModal 一致）。
- Access `user_info.company_member_id` / collaborators `.id` 用于人过滤。
- 组成员 API 返回 `user`（user_id）→ 用 collaborators 的 `user`→`id` 映射。

## 5. API（全部复用，无新增）

| 用途 | 方法 | 路径 |
|------|------|------|
| 可访问人+小组列表 | GET | `/api/tenant/{t}/projects/workspace-access/workspace-permissions/?workspace_id=` |
| 协作人池（user→member 映射） | GET | `/api/tenant/{t}/projects/workspace-access/workspace-collaborators/?workspace_id=` （WorkPanel 已拉） |
| 小组成员 | GET | `/api/tenant/{t}/accounts/groups/{groupId}/members/` |

## 6. 前端落点

| 文件 | 职责 |
|------|------|
| `utils/workPanelAccessFilter.js` | normalize / chip 文案 / `filterTodosByAccess` / 组员 ID 解析 |
| `composables/useWorkPanelAccessFilter.js`（可选） | 加载 permissions、选中态、拉组员 |
| `WorkPanelHeader.vue` | 「人/小组」下拉 + 清除 chip（`data-alias=access-filter-*`） |
| `WorkPanel.vue` | 将 access 过滤接入 `filteredTodos` 流水线（在机器过滤之后或之前均可，须文档化 AND） |

### UI 行为

- 控件：分段「人 | 小组」+ 可搜索下拉；选中后 Header 旁 chip「过滤：{名称}」。
- 再次选同一项或点 chip × → 清除。
- 切换工作空间 → 清除选中并重载选项。
- 加载失败 → `console.warn` + 空选项，不打断看板。

## 7. 日志

- 前端：选项加载失败 warn（不记 PII 全量列表）；选中变更可 debug：`kind` + `id`（禁止 dump 邮箱）。
- 后端：无新路径；既有 access/groups 日志不变。

## 8. 测试

- 单元：`workPanelAccessFilter.test.js` — 人/组匹配、空组、与无 owner 任务、ID 字符串化。
- Playwright：`WorkPanel.accessFilter.playwright.test.js` — 打开下拉、选人后卡片减少、清除恢复（可 mock API）。

## 9. 架构交付物

**本迭代不创建新架构版本**：无新服务组件、无新 Rel_Flow、无新表所有权变更。基线仍为 v27 ✅ current；并行 target（v26/v28 等）不受影响。

## 10. 关键决策摘要

| 决策 | 理由 |
|------|------|
| 客户端过滤 | 同 005；零后端改动；延迟低 |
| 选项=workspace access | 用户明确「可以访问当前工作空间的人或小组」 |
| 匹配 owner∪assignees | 覆盖指派与协作，符合「相关任务」直觉 |
| 不持久化 | 未要求；与机器过滤一致，降低 payload 耦合 |
| 组员经既有 members API | 无新接口；Django 存量 ViewSet |
