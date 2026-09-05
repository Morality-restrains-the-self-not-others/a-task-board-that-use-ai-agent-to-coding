# 实施计划：闲置复用启动保护 + 孤儿交叉校验

- **日期**: 2026-07-22

## Tasks

- [x] T1 红灯：单测 boot 无 idle_since 不 reuse；真 idle 可 reuse；orphan skip_owned
- [x] T2 绿灯 A：`findIdleMachineForReuseExcluding` 读 `idle_since` + Starting 排除 + skip 日志
- [x] T3 绿灯 C：`reconcileOrphanInstancesByName` owned 集合 + skip 日志
- [x] T4 更新既有 `TestStartVmAutoPreferIdleReuseReturnsReuse` 写入 idle_since
- [x] T5 `go test` taskCloudService 相关包全绿
- [x] T6 更新设计文档架构章节路径；OPT 落盘

## 事件契约任务

- [x] 无新事件 — 意图文档已书面例外
