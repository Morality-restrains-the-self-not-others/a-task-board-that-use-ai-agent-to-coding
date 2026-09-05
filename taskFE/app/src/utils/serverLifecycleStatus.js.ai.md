# Companion：`serverLifecycleStatus.js`

## 扩展契约（强制）

新增服务器生命周期展示态时：

1. **只改**本模块（`serverLifecycleStatus.js`）与 `serverLifecycleStatus.test.js`
2. **禁止**为了加新态去改 `TaskDetailServerStartStatusPanel.vue`（面板只绑定 `resolveServerLifecycleLabel` / `serverLifecycleDotClass` / `serverLifecycleTextClass`）

### 改动清单

| 场景 | 改哪里 |
|------|--------|
| 新后端 `serverStatus` 码映射到已有文案 | `SERVER_STATUS_CODE_TO_LABEL`（`maps every … entry` 测例自动覆盖） |
| 新展示文案（如「迁移中」「等待容器」） | `SERVER_LIFECYCLE_STYLES` + `ServerLifecycleLabel` typedef + 码表 + 测例中的 styles key 列表 |
| 运行中/启动中布尔优先逻辑变更 | `resolveServerLifecycleLabel` + 测例 |
| VM Running 但容器未登记 | starting + `runtimeStatus=Running` → 「等待容器」 |
| 超时收口后 VM 仍 Running | `serverStatus=error` 优先于 runtime Running → 「启动失败」 |

### 自检

- 跑：`npm test -- src/utils/serverLifecycleStatus.test.js`
- 契约测例会断言：码表每个 label 都有样式；样式表每个 label 都有 helper 输出
