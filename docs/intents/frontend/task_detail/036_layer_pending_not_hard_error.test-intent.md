# 测试意图：任务详情文件树 / 执行日志等待态

## 对应意图
`036_layer_pending_not_hard_error.intent.md`

## 测点

| ID | 测点 | 类型 | 位置 |
|---|---|---|---|
| T1 | `layer not found` / 409 缺业务地址 → waiting 文案 | 单元 | `containerComputeWaiting.test.js` |
| T2 | 文件树 404 layer not found 展示 waiting 节点且带 data-traceId | 单元 | `TaskDetailProjectFileTree.layer-pending.unit.test.js` |
| T3 | `containerEndpointRegistered=false` 仍拉取 clone-log，不写旧红错文案 | 单元 | `taskDetailExecLog.test.js` |
| T4 | 409 映射为等待文案而非 topError 硬错误 | 单元 | `taskDetailExecLog.test.js` |

## 成功标准
T1–T4 vitest 全绿。
