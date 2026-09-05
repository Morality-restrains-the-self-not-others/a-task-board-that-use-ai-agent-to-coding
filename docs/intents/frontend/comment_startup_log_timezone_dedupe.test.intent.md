# 测试意图：评论启动日志时区与去重

## 测试目标

验证启动日志不会因 SSE + list 双通道、UTC/本地时钟、`trace_id=` 后缀而重复。

## 测试分层

- 单元：`bindingServerStartupLogs.js`、`createBindingStatusLogs.js`
- 组合：`useCommentContainerBindings.test.js`（refresh 合并后端日志）

## 用例矩阵

### T1 — 去重忽略时钟与 trace_id 后缀

- **给定** binding 行 `[23:30:54] [i-abc、task_t1_c1] 容器运行时安装完成`
- **并且** 任务级行 `[15:30:53] [i-abc、task_t1_c1] 容器运行时安装完成 trace_id=bb1157e1b936e3d8eb7e7b05`
- **当** `mergeBindingAndServerStartupLogs`
- **则** 结果 1 行，且包含 `trace_id=`

### T2 — naive UTC 与 RFC3339 Z 同一瞬间

- **给定** `created_at` 为 `2026-08-13 15:30:59` 与 `2026-08-13T15:30:59Z`
- **当** `parseBindingLogDate`
- **则** `toISOString()` 均为 `2026-08-13T15:30:59.000Z`

### T3 — hydrate 不重复追加已有 SSE 行

- **给定** 本地已有无后缀 UserData 行
- **当** `mergeBackendBindingLogs` 灌入同文案 + `trace_id=`
- **则** 行数不变（或替换为带 suffix 的一行），不得变成两行

### T4 — 不同步骤不去重

- **给定** 「容器运行时安装完成」与「容器运行时服务已就绪」
- **当** merge
- **则** 仍为两行

## 数据与环境

- 不依赖浏览器时区；T2 用 `toISOString()`。

## 通过标准

- 上列 vitest 全绿。

## 业务意图 → 事件对照

纯前端展示，无新事件；见功能意图例外理由。

## 自动化落点

- `taskFE/app/src/utils/bindingServerStartupLogs.test.js`
- `taskFE/app/src/composables/taskDetail/createBindingStatusLogs.test.js`
- `taskFE/app/src/composables/taskDetail/useCommentContainerBindings.test.js`
