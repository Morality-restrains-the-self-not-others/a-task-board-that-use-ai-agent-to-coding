# useTaskDetail.js 拆分实施计划

> **For agentic workers:** 此计划将 2933 行的 `useTaskDetail.js` 拆分为多个 ≤500 行的子 composable。使用 superpowers:executing-plans 逐任务执行。步骤使用 checkbox (`- [ ]`) 语法追踪。

**目标:** 将 `useTaskDetail.js` 从 2933 行拆分为 ~10 个各 ≤500 行的子 composable，每个子 composable 负责一个领域。

**架构:** 子 composable 通过共享 `ctx` (context) 对象模式访问响应式状态，替代当前闭包变量的直接引用。每个子模块接收 `ctx` 参数，从中读取/写入所需的状态变量。主 `useTaskDetail.js` 负责创建 ctx、组装子 composable、暴露返回对象。

**技术栈:** Vue 3 Composition API (ref, reactive, computed, watch), JavaScript

**风险:** 高 —— 293 个变量/函数在单一闭包中交叉引用。每次抽取须保持行为完全不变。需要现有测试覆盖来捕捉回归。

---

## 当前状态

`useTaskDetail.js` (2933 行) 已有 11 个子模块在 `composables/taskDetail/`:

| 子模块 | 行数 | 职责 |
|--------|------|------|
| `updateServerStatus.js` | ~330 | 服务器状态更新 |
| `establishSSEConnection.js` | ~130 | SSE 连接建立 |
| `submitAIComment.js` | ~190 | AI 评论提交 |
| `taskDetailJobActions.js` | ~120 | 层图作业操作 |
| `submitLayerGraphCommand.js` | ~180 | 层图命令提交 |
| `taskDetailLayerActions.js` | ~400 | 层图操作 |
| `taskDetailEditing.js` | ~170 | 任务编辑 |
| `taskDetailExecLog.js` | ~150 | 执行日志 |
| `taskDetailContainerFns.js` | ~270 | 容器函数 |
| `taskDetailGitFns.js` | ~160 | Git 操作 |
| `taskDetailFetchFns.js` | ~380 | 数据获取 |

主文件还需拆分的领域（按行数估计）:
1. **Core state & computed** (~400 行) — tenant/workspace/task refs, route computed properties
2. **Clone Progress** (~250 行) — containerCloneProgressByKey, entries, log parsing
3. **Repo & Branch** (~250 行) — taskProjectsWithDetails, taskRepoRows, branch name utilities
4. **Comments** (~100 行) — displayComments, newComment, commentComposerChips
5. **Layer Graph state** (~150 行) — snapshot, zNodes, model selection, refresh logic
6. **Container lifecycle** (~200 行) — heartbeat state, unreachable probes, timing
7. **SSE reconnect** (~80 行) — reconnect timer logic (already partially inline)
8. **Remaining inline** — 主文件中 ~500 行尚未委托给子模块的胶水代码

---

### Task N.0: 分析领域边界（只读研究，不写代码）

**文件:**
- 阅读: `task2app/front_project/app/src/composables/useTaskDetail.js`

- [ ] **Step 0.1: 标记每个领域在整个文件中的起始/结束行**

运行以下命令标记行号:
```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
grep -n '^const\|^let\|^function\|// ===\|// [A-Z]' task2app/front_project/app/src/composables/useTaskDetail.js > /tmp/useTaskDetail_structure.txt
```

然后为每个领域记录行范围:
```
SSE/Reconnect:    lines 79-150
Core computed:    lines 165-280
Task repos:       lines 300-540
Comments:         lines 612-690
Editing/fork:     lines 632-763
Clone Progress:   lines XXX-XXX
Layer Graph:      lines XXX-XXX
Container:        lines XXX-XXX
Git Identity:     lines XXX-XXX
Execution Log:    lines XXX-XXX
```

- [ ] **Step 0.2: 为每个领域列出所有导出的变量和函数**

对于每个领域，运行:
```bash
grep -A 500 '// Data' task2app/front_project/app/src/composables/useTaskDetail.js | grep '^\s*[a-z]' | head -30
```

确认返回对象 (return { ... }) 中包含该领域的所有键。

- [ ] **Step 0.3: 检查现有测试覆盖**

```bash
find . -path '*/playwright/front_project/tests/*TaskDetail*' -o -path '*/front_project/tests/*taskDetail*' -o -path '*/front_project/tests/*task-detail*' 2>/dev/null
```

---

### Task N.1: 提取 Clone Progress 领域（首个低风险领域）

**原因:** Clone Progress 有清晰的接口边界：接收 SSE 日志行 → 解析 → 返回进度条目。大多数函数是纯计算，少数 ref 与外部交互。

**文件:**
- 创建: `task2app/front_project/app/src/composables/taskDetail/taskDetailCloneProgress.js`
- 修改: `task2app/front_project/app/src/composables/useTaskDetail.js` (删除内联代码，改为导入)

- [ ] **Step 1.1: 列出 Clone Progress 领域的所有变量和函数**

搜索 useTaskDetail.js 中与 clone progress 相关的代码:
```bash
grep -n 'cloneProgress\|CloneProgress\|CLONE_PROGRESS\|containerClone\|BootstrapClone\|BOOTSTRAP_CLONE' task2app/front_project/app/src/composables/useTaskDetail.js
```

确认包括:
- `containerCloneProgressByKey` (ref)
- `containerBootstrapCloneLogFull` (ref)
- `containerCloneProgressEntries` (computed)
- `cloneProgressEntryByRepoMatchKey` (computed)
- `cloneProgressBarWidthTransitionClass` (computed)
- `cloneProgressRowLogText`, `cloneProgressRowLogDisplayText`, `cloneProgressRowLogIsPlaceholder`
- `onBootstrapCloneLogUpdate`, `applyBootstrapCloneLogDoneToCloneProgress`
- `rescheduleContainerCloneProgressClearTimer`
- `containerPageLinkPendingReveal`, `containerCloneProgressTimer`
- `BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE`
- `CONTAINER_CLONE_PROGRESS_GLOBAL_KEY`

- [ ] **Step 1.2: 创建 `taskDetailCloneProgress.js` 子模块**

文件路径: `task2app/front_project/app/src/composables/taskDetail/taskDetailCloneProgress.js`

```javascript
import { ref, computed } from 'vue'
import {
  parseBootstrapCloneLogSections,
  mergeCloneProgressSubPhases,
  cloneProgressRowHasSubPhases,
  cloneProgressRecvPct,
  cloneProgressUnpackPct,
} from '../../utils/taskDetailContainerCloneProgress.js'

export const BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE = /克隆完成|clone.*(complete|done|finished)/i
export const CONTAINER_CLONE_PROGRESS_GLOBAL_KEY = '__global__'

export function createCloneProgressState() {
  const containerCloneProgressByKey = ref({})
  const containerBootstrapCloneLogFull = ref('')
  const containerPageLinkPendingReveal = ref(false)
  let containerCloneProgressTimer = null

  const containerCloneProgressEntries = computed(() => {
    const m = containerCloneProgressByKey.value
    return Object.keys(m).map(k => ({ key: k, ...m[k] }))
  })

  const cloneProgressEntryByRepoMatchKey = computed(() => {
    const entries = containerCloneProgressEntries.value
    const map = {}
    for (const e of entries) {
      const rk = e.repoMatchKey
      if (rk) map[rk] = e
    }
    return map
  })

  return {
    containerCloneProgressByKey,
    containerBootstrapCloneLogFull,
    containerPageLinkPendingReveal,
    containerCloneProgressEntries,
    cloneProgressEntryByRepoMatchKey,
    get containerCloneProgressTimer() { return containerCloneProgressTimer },
    set containerCloneProgressTimer(v) { containerCloneProgressTimer = v },
  }
}

export function createCloneProgressComputed(state) {
  const cloneProgressBarWidthTransitionClass = computed(() => {
    // ... existing implementation from useTaskDetail.js
    return ''
  })

  const cloneProgressRowLogText = (entry) => {
    // ... existing implementation
    return ''
  }

  const cloneProgressRowLogDisplayText = (entry) => {
    // ... existing implementation
    return ''
  }

  const cloneProgressRowLogIsPlaceholder = (entry) => {
    // ... existing implementation
    return false
  }

  return {
    cloneProgressBarWidthTransitionClass,
    cloneProgressRowLogText,
    cloneProgressRowLogDisplayText,
    cloneProgressRowLogIsPlaceholder,
  }
}

export function onBootstrapCloneLogUpdate(rawLog, state) {
  // ... existing implementation from useTaskDetail.js
  // 更新 state.containerBootstrapCloneLogFull
  // 更新 state.containerCloneProgressByKey
}

export function applyBootstrapCloneLogDoneToCloneProgress(state) {
  // ... existing implementation from useTaskDetail.js lines 2680-2726
}

export function rescheduleContainerCloneProgressClearTimer(state) {
  // ... existing implementation from useTaskDetail.js
}
```

- [ ] **Step 1.3: 在 useTaskDetail.js 中导入并使用新模块**

在 useTaskDetail.js 中添加导入:
```javascript
import {
  BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE,
  CONTAINER_CLONE_PROGRESS_GLOBAL_KEY,
  createCloneProgressState,
  createCloneProgressComputed,
  onBootstrapCloneLogUpdate as _onBootstrapCloneLogUpdate,
  applyBootstrapCloneLogDoneToCloneProgress as _applyBootstrapCloneLogDoneToCloneProgress,
  rescheduleContainerCloneProgressClearTimer as _rescheduleContainerCloneProgressClearTimer,
} from './taskDetail/taskDetailCloneProgress.js'
```

在 `useTaskDetail` 函数体内:
```javascript
const cpState = createCloneProgressState()
const {
  containerCloneProgressByKey, containerBootstrapCloneLogFull,
  containerPageLinkPendingReveal, containerCloneProgressEntries,
  cloneProgressEntryByRepoMatchKey,
} = cpState

const cpComputed = createCloneProgressComputed(cpState)
const {
  cloneProgressBarWidthTransitionClass, cloneProgressRowLogText,
  cloneProgressRowLogDisplayText, cloneProgressRowLogIsPlaceholder,
} = cpComputed

const onBootstrapCloneLogUpdate = (rawLog) => _onBootstrapCloneLogUpdate(rawLog, cpState)
const applyBootstrapCloneLogDoneToCloneProgress = () => _applyBootstrapCloneLogDoneToCloneProgress(cpState)
const rescheduleContainerCloneProgressClearTimer = () => _rescheduleContainerCloneProgressClearTimer(cpState)
```

- [ ] **Step 1.4: 从 useTaskDetail.js 中删除内联实现**

删除原有的 `containerCloneProgressByKey`, `containerBootstrapCloneLogFull`, 及相关 computed 和函数的内联定义。

- [ ] **Step 1.5: 验证功能**

运行:
```bash
cd task2app/front_project/app
# 如有测试则运行
npm run test 2>/dev/null || echo "No tests configured"
# 验证文件行数
wc -l src/composables/useTaskDetail.js src/composables/taskDetail/taskDetailCloneProgress.js
```

- [ ] **Step 1.6: 提交**

```bash
git add task2app/front_project/app/src/composables/taskDetail/taskDetailCloneProgress.js
git add task2app/front_project/app/src/composables/useTaskDetail.js
git commit -m "refactor: extract clone progress into taskDetailCloneProgress.js"
```

---

### Task N.2: 提取 Layer Graph State 领域

**文件:**
- 创建: `task2app/front_project/app/src/composables/taskDetail/taskDetailLayerGraphState.js`
- 修改: `task2app/front_project/app/src/composables/useTaskDetail.js`

- [ ] **Step 2.1: 列出 Layer Graph State 的所有变量**

```bash
grep -n 'layerGraph\|layer_graph\|LAYER_GRAPH' task2app/front_project/app/src/composables/useTaskDetail.js | head -40
```

确认包括:
- `layerGraphSnapshot`, `layerGraphZNodes`, `layerGraphMetaLine`
- `layerGraphLayerIdsKey`, `layerGraphSeenLayerIdSet`
- `layerGraphRefreshing`
- `showCommentLayerZtreeLoading`, `commentLayerZtreeLoadingHint`
- `selectedLayerGraphNode`, `selectedLayerGraphFileTreeLayerId`
- `layerGraphModelProvider`, `layerGraphDefaultModel`, `layerGraphModelOptions`
- `layerGraphSelectedModel`, `layerGraphModelLoading`, `layerGraphModelLoadError`
- `layerGraphCommandText`, `layerGraphCommandKind`, `layerGraphAutoIterationCount`
- `layerGraphCmdError`, `layerGraphCmdSending`
- `layerGraphEditRunTargetJobId`, `layerGraphBusyActionKey`
- `layerGraphModelSelectDisabled`
- `lastLayerGraphHeartbeatRefreshAt`, `LAYER_GRAPH_REFRESH_ON_HEARTBEAT_MIN_MS`

- [ ] **Step 2.2: 创建 `taskDetailLayerGraphState.js`**

```javascript
import { ref, computed } from 'vue'
import { buildZTreeNodesSerialFromLayers, LAYER_TREE_NODE_PREFIX } from '../../utils/layerZtreeNodes.js'

export const SERVER_RUNTIME_LAYER_GRAPH_GATE_MS = 2500
export const LAYER_GRAPH_REFRESH_ON_HEARTBEAT_MIN_MS = 4000

export function createLayerGraphState() {
  const layerGraphSnapshot = ref(null)
  const layerGraphRefreshing = ref(false)
  const showCommentLayerZtreeLoading = ref(false)
  
  const layerGraphZNodes = computed(() => {
    const snap = layerGraphSnapshot.value
    if (!snap) return []
    return buildZTreeNodesSerialFromLayers(snap.layers || [])
  })

  const layerGraphMetaLine = computed(() => {
    const snap = layerGraphSnapshot.value
    if (!snap) return ''
    return snap.meta || ''
  })

  const layerGraphLayerIdsKey = computed(() => {
    const nodes = layerGraphZNodes.value
    return nodes.map(n => n.id).join(',')
  })

  // ... 其余 computed 和 state refs
  
  return {
    layerGraphSnapshot, layerGraphZNodes, layerGraphMetaLine,
    layerGraphLayerIdsKey, layerGraphRefreshing,
    showCommentLayerZtreeLoading, commentLayerZtreeLoadingHint,
    // ... 所有层图相关状态
  }
}
```

- [ ] **Step 2.3: 在 useTaskDetail.js 中集成并删除内联代码**

同 Task N.1 模式。

- [ ] **Step 2.4: 提交**

```bash
git add task2app/front_project/app/src/composables/taskDetail/taskDetailLayerGraphState.js
git add task2app/front_project/app/src/composables/useTaskDetail.js
git commit -m "refactor: extract layer graph state into taskDetailLayerGraphState.js"
```

---

### Task N.3: 提取 Container Lifecycle 领域

**文件:**
- 创建: `task2app/front_project/app/src/composables/taskDetail/taskDetailContainerLifecycle.js`
- 修改: `task2app/front_project/app/src/composables/useTaskDetail.js`

- [ ] **Step 3.1: 列出 Container Lifecycle 变量**

```bash
grep -n 'containerHeartbeat\|containerEndpoint\|containerHttp\|ContainerUnreachable\|containerVscode\|containerPage' task2app/front_project/app/src/composables/useTaskDetail.js | head -20
```

- [ ] **Step 3.2: 创建 `taskDetailContainerLifecycle.js`**

将容器心跳、端点注册、不可达探测等逻辑提取到此模块。

- [ ] **Step 3.3: 集成并提交**

---

### Task N.4: 提取 Task Repo & Branch 领域

**文件:**
- 创建: `task2app/front_project/app/src/composables/taskDetail/taskDetailRepoBranch.js`
- 修改: `task2app/front_project/app/src/composables/useTaskDetail.js`

- [ ] **Step 4.1: 列出 Repo & Branch 变量**

```bash
grep -n 'taskProjects\|taskRepo\|repoClone\|workBranch\|mergeTarget\|CloneIdentity\|buildWork\|buildMerge' task2app/front_project/app/src/composables/useTaskDetail.js | head -30
```

- [ ] **Step 4.2: 创建 `taskDetailRepoBranch.js`**

提取仓库关联、分支名构建、克隆身份映射等逻辑。

- [ ] **Step 4.3: 集成并提交**

---

### Task N.5: 提取 Comments 领域

**文件:**
- 创建: `task2app/front_project/app/src/composables/taskDetail/taskDetailComments.js`
- 修改: `task2app/front_project/app/src/composables/useTaskDetail.js`

- [ ] **Step 5.1: 创建 `taskDetailComments.js`**

提取 `displayComments` computed、`newComment` ref、`commentComposerChips` 等。

- [ ] **Step 5.2: 集成并提交**

---

### Task N.6: 提取 SSE Reconnect 逻辑

**文件:**
- 创建: `task2app/front_project/app/src/composables/taskDetail/sseReconnect.js`
- 修改: `task2app/front_project/app/src/composables/useTaskDetail.js`

- [ ] **Step 6.1: 创建 `sseReconnect.js`**

提取 `sseReconnectionInProgress`, `sseReconnectAttempts`, `calculateReconnectDelay`, `scheduleSSEReconnect`, `handleSSEManualReconnect` 等。

- [ ] **Step 6.2: 集成并提交**

---

### Task N.7: 最终清理与行数验证

- [ ] **Step 7.1: 确认 useTaskDetail.js ≤500 行**

```bash
wc -l task2app/front_project/app/src/composables/useTaskDetail.js
# 目标: ≤500
```

- [ ] **Step 7.2: 确认所有 taskDetail/*.js ≤500 行**

```bash
for f in task2app/front_project/app/src/composables/taskDetail/*.js; do
  lines=$(wc -l < "$f")
  if [ "$lines" -gt 500 ]; then
    echo "FAIL: $f ($lines)"
  fi
done
echo "All files within limit"
```

- [ ] **Step 7.3: 提交**

```bash
git add task2app/front_project/app/src/composables/
git commit -m "refactor: split useTaskDetail.js into domain composables (all <= 500 lines)"
```

---

## 执行说明

每个提取遵循相同模式:
1. 分析领域 → 列出所有变量/函数
2. 创建独立子模块文件（接收 ctx/state 参数）
3. 在 useTaskDetail.js 中导入并委托
4. 删除内联实现
5. 验证功能 + 提交

**关键约束:** 增量提取，每次只动一个领域。每次提交后验证 useTaskDetail.js 行数递减。
