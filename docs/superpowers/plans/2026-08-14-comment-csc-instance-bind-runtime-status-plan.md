# Plan: 评论 CSC 启动成功必须绑定 instance

- **状态**: in progress
- **设计**: `docs/superpowers/specs/2026-08-14-comment-csc-instance-bind-runtime-status-design.md`

## 任务清单

- [x] T1 persist 写 instance + Starting；0 行返回 error
- [x] T2 无 comment/csc 不得写任务级；返回 error
- [x] T3 executeStartVmNative：空 instance → error
- [x] T4 upsert 空 instance+空 status+comment 保留旧 i-
- [x] T5 persist 公网 IP → Running
- [x] T6 runtime-status：有 instance 不报「等待分配」；空 instance heal
- [x] T7 boot-progress：有 i- 且 Starting → Running
- [x] 意图文档已有
- [x] 精准编译重启 task-cloud-service（已登记，待 9999 执行）

## 事件契约

无新 Kafka。绑定/升 Running 为同库投影（见意图例外）。
