# Step 9 审查：runAll 编排器独立

- **Date:** 2026-08-23
- **CRG:** unavailable for Go runAll（图语言无 Go）

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | SIGTERM 默认 keep；Setsid 不可叠加 Setpgid（EPERM 已测）；T4 cancel 后 PID 仍存活 |
| Readability | 策略在 domain；EPERM 注释在 helper |
| Architecture | 不把策略堆进 4714 行 runner.go；stop-all 语义未改 |
| Security | 无新公网口；`RUNALL_SHUTDOWN_SERVICES` 仅本机 env |
| Performance | 无热路径变化 |

## 安全清单（本增量适用项）

- [x] 无密钥入代码/日志
- [x] 无新用户输入面
- [x] 无 SQL

## 发现

无 Critical / Required。Nit：`runner.go` 仍为遗留巨石，未在本增量削到 500 行。
