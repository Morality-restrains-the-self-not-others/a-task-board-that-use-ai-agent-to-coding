# 权限分析：排队调度自动关闭

- 日期：2026-07-22
- 设计：`2026-07-22-schedule-rhythm-auto-close-design.md`

## 触点矩阵

| 触点 | 角色 | 资源 | 操作 | 既有鉴权 | 结论 |
|------|------|------|------|----------|------|
| PATCH `schedule_rhythm.auto_close` | 工作空间可写成员 | 顶层任务节奏 | write | 与 todos PATCH 一致 | ✅ 不新增角色 |
| GET 回显 | 工作空间可读成员 | 同上 | read | 既有 | ✅ |
| closing-soon / shutdown 代调 | TTS internal → Cloud → Gateway | 槽位任务容器 | invoke | InternalSecret / Gateway secret | ✅ 客户端不可伪造批量释放 |
| stop-vm 回退 | TTS → Cloud | 槽位任务机器 | write | InternalSecret | ✅ |

## CRG 触点

- graph_status: sparse/unavailable for Go TTS；文件级触点见设计 🕸️
- 敏感边：TTS→Cloud stop-vm / lifecycle；须 internal，禁止浏览器直传批量 auto_close 释放旁路

## 风险

| 风险 | 缓解 |
|------|------|
| 伪造 started_via 扩大释放面 | 仅清 `queued_machine_slots`，不扫全 CSC |
| 重复释放 | warn_key / release_key 幂等 |
