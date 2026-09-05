# 设计：历史服务器启动记录 — 运行时长与启停原因

日期：2026-07-19  
入口：goal-mode（跳过 USER GATE）

## 问题

任务详情历史启动记录卡片仅展示启动时间/平台/实例/硬件，缺少单次会话的运行持续时间与启停原因，运维无法快速对照。

## 方案（已采用）

**纯前端增强**：在 `ServerConfigServerStartHistoryPanel` 展示既有 API 字段。

| UI | 数据 | 说明 |
|----|------|------|
| 运行持续时间 | `started_at`→`stopped_at`（或 now） | 复用 `formatUptimeSinceMs`；起点缺省 `created_at` |
| 启动原因 | `runtime_source` | 领域无独立 `start_reason`；`runtime_source` 即启动路径语义 |
| 关闭原因 | `stop_reason` | API 已下发，仅补展示与中文标签 |

## 否决方案

1. **新增 `start_reason` 列**：需迁移与全链路写入，超出页面元素调整范围。
2. **后端计算 `duration_seconds`**：前端已有时长格式化，避免重复契约。

## 权限 / NFR / DDD（裁剪）

- 权限：只读展示，无新 endpoint。
- NFR：L2；标签映射为纯函数，无额外请求。
- DDD：不新增实体；展示映射属 UI 适配层。

## 实施要点

1. `serverStartHistoryDisplay.js` + 单测
2. 更新历史面板模板三行
3. intent `033_*`；公网 SPA `runall-lifecycle.sh build`
