# 任务详情「仓库地址已变更」同步按钮缺失修复设计

日期：2026-05-27  
状态：待批准

## 问题

用户在任务详情页「关联项目」区域看到「仓库地址已变更」徽章，但找不到「同步/更新任务仓库地址」按钮，无法将 `TaskProject.repo_address` 与项目当前 `git_repos` 对齐。

复现 URL（示例）：

`/tenant/827923618468040704/workspace/827923618602258432/task-detail/846269443533955072/?relayToTrae=true`

## 根因分析

此前设计（`2026-05-27-task-detail-stale-repo-address-clone-design.md`）分两部分交付：

| 区域 | 预期行为 | 实际状态 |
|------|----------|----------|
| 关联项目卡片 | 显示「仓库地址已变更」徽章 | ✅ 已实现（`TaskDetailLinkedProjectsPanel` + `useTaskDetail`） |
| 直接启动面板 | 显示不一致横幅 +「更新任务仓库地址」/「不更新，继续启动」+ 阻断启动 | ❌ UI 组件已写好，父组件未接线 |

具体缺口：

1. `ServerConfigRelayDirectPanel.vue` 已具备 `staleRepoMismatches`、`syncStaleRepoAddresses` 等 props/事件。
2. `ServerConfig.logic.vue` **未**传入上述 props，**未**绑定 `@syncStaleRepoAddresses` / `@acknowledgeStaleRepo`。
3. `taskRepoAddressMismatch.js`（`collectTaskRepoAddressMismatches` / `syncTaskRepoAddressesFromProjects`）**未被任何文件 import**。
4. Playwright 回归测试 `TaskDetail.relay-to-trae-stale-repo-address.playwright.test.js` 已编写，但因接线缺失会在真实环境失败。

用户在「关联项目」找按钮是合理预期——徽章与操作分离导致困惑。

## 价值流影响

受影响流（`value-stream.yaml`）：

- **任务协作 / relay-to-trae 直接启动**：启动前需检测陈旧仓库地址并允许同步。
- **项目与工作空间 / 任务-项目关联**：`TaskProject.repo_address` 与 `ProjectRepo` 一致性。

需更新/补充测试：

- `playwright/front_project/tests/TaskDetail.relay-to-trae-stale-repo-address.playwright.test.js`（已有，修复后应通过）
- 新增单元测试：`ServerConfig.logic` 或 `taskRepoAddressMismatch` 接线逻辑
- 可选：关联项目面板同步按钮的前端单测

## 领域概念（供 DDD 步骤参考）

- **Bounded Context**：任务协作（Task）、项目仓库（ProjectRepo）
- **实体**：TaskProject（含 `repo_address`）、Project（含 `git_repos`）
- **值对象**：RepoAddressMismatch（stored vs current）
- **领域事件**：TaskRepoAddressSynced（PATCH 成功后任务记录与项目对齐）

## 方案对比

### 方案 A：仅修复「直接启动」面板接线（最小修复）

在 `ServerConfig.logic.vue` 接入 `taskRepoAddressMismatch.js`，将 mismatch 状态传给 `ServerConfigRelayDirectPanel`。

- 优点：改动面小，与现有 Playwright 测试一致。
- 缺点：用户仍在「关联项目」看到徽章但无操作，需切到「直接启动」tab 才能同步。

### 方案 B：接线 + 在关联项目卡片增加同步按钮（推荐）

在方案 A 基础上，于 `TaskDetailLinkedProjectsPanel` 徽章旁增加「同步仓库地址」按钮（仅 `repo_address_mismatch === true` 时显示），复用同一 PATCH 逻辑（经 `useTaskDetail` 暴露 handler 或 emit 至 `TaskDetail.vue`）。

- 优点：徽章与操作同处，符合用户心智；relay 面板保留完整横幅与「不更新，继续启动」流程。
- 缺点：两处入口需共用逻辑，避免状态不同步。

### 方案 C：仅在关联项目放按钮，relay 面板只阻断不展示同步

- 优点：单一入口。
- 缺点：与已写 Playwright 测试及 relay 面板 UI 冲突，需改测试。

**推荐方案 B**。

## 详细设计（方案 B）

### 1. ServerConfig.logic.vue — relay 面板接线

```javascript
import {
  collectTaskRepoAddressMismatches,
  syncTaskRepoAddressesFromProjects,
} from '../utils/taskRepoAddressMismatch.js'

// computed
const staleRepoMismatches = computed(() =>
  collectTaskRepoAddressMismatches(props.task?.projects))

const staleRepoAcknowledged = ref(false)

const startBlockedByStaleRepo = computed(() =>
  staleRepoMismatches.value.length > 0 && !staleRepoAcknowledged.value)

// watch：task.projects 变化时重置 acknowledged
watch(() => props.task?.projects, () => { staleRepoAcknowledged.value = false }, { deep: true })

const staleRepoSyncLoading = ref(false)

async function syncStaleRepoAddresses() {
  // PATCH todos，emit('task-updated', data)，可选调用 props.refreshTaskDetail
}

function acknowledgeStaleRepo() {
  staleRepoAcknowledged.value = true
}
```

模板补充：

```vue
<ServerConfigRelayDirectPanel
  ...
  :stale-repo-mismatches="staleRepoMismatches"
  :stale-repo-sync-loading="staleRepoSyncLoading"
  :start-blocked-by-stale-repo="startBlockedByStaleRepo"
  @sync-stale-repo-addresses="syncStaleRepoAddresses"
  @acknowledge-stale-repo="acknowledgeStaleRepo"
/>
```

`syncStaleRepoAddresses` 使用 `syncTaskRepoAddressesFromProjects`，成功后 `emit('task-updated', patchedTask)` 更新 `localTask`，使徽章与横幅同步消失。

### 2. useTaskDetail.js + TaskDetailLinkedProjectsPanel — 关联项目同步按钮

- 在 `useTaskDetail` 新增 `syncTaskRepoAddressForProject(projectId)` 或 `syncAllStaleTaskRepoAddresses()`。
- 复用 `syncTaskRepoAddressesFromProjects`（传入 `localTask.value.projects`）。
- PATCH 成功后更新 `localTask`，`taskProjectsWithDetails` 自动刷新。
- 在徽章旁增加按钮：

```vue
<button
  v-if="!isEditing && tp.repo_address_mismatch"
  data-testid="task-repo-address-sync-btn"
  @click="emit('sync-repo-address', tp.project_id)"
>
  同步仓库地址
</button>
```

- `TaskDetail.vue` 绑定 `@sync-repo-address="syncStaleTaskRepoAddress"`。

### 3. 启动阻断

- relay「启动」按钮：`startBlockedByStaleRepo` 为 true 时 disabled（已有 prop，接线即可）。
- 用户点击「不更新，继续启动」后 `staleRepoAcknowledged = true`，允许启动（容器侧仍按项目当前 URL 克隆）。

### 4. 非目标

- 不修改后端 `repo_address_mismatch` 字段语义。
- 不自动静默 PATCH；必须用户显式点击同步或确认不更新。

## 测试计划

1. **单元**：`collectTaskRepoAddressMismatches` 边界（空 projects、无 mismatch）。
2. **组件**：`ServerConfig.logic` 或 extract composable — mismatch 时 `startBlockedByStaleRepo === true`。
3. **E2E**：现有 `TaskDetail.relay-to-trae-stale-repo-address.playwright.test.js` 全绿。
4. **E2E（可选）**：关联项目面板点击「同步仓库地址」后徽章消失。

## 验收标准

- [ ] 「关联项目」出现 mismatch 徽章时，同卡片可见「同步仓库地址」按钮，点击后 PATCH 成功、徽章消失。
- [ ] 「直接启动」tab 显示橙色不一致横幅及「更新任务仓库地址」「不更新，继续启动」。
- [ ] 未确认前「启动」按钮 disabled；确认或同步后可启动。
- [ ] Playwright stale-repo 回归测试通过。
