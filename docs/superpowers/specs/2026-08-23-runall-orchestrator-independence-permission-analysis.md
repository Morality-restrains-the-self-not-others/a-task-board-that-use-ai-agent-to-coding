# 角色权限分析：runAll 编排器生命周期解耦

- **Date:** 2026-08-23
- **Design:** `docs/superpowers/specs/2026-08-23-runall-orchestrator-independence-design.md`

## 结论

无新租户 API、无新 RBAC 角色。变更在 **本机 Status UI（:9999）运维面**，主体是操作员/Agent，不是租户用户。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| SIGTERM/SIGINT → 保留托管服务 | 主机操作员 / 内核信号 | System（本机进程） | 编排器退出 | 无对外 HTTP | ✅ 充分 | 不新增鉴权 |
| `RUNALL_SHUTDOWN_SERVICES=1` | 本机环境变量 | System | 调试拆栈 | 仅本机进程可见 | ✅ 充分 | 禁止写入 conf 提交为 true |
| POST `/api/shutdown-self` | 本机/新 runAll 实例 | System | 热替换 | 已监听 9999，无租户 token | ✅ 充分 | 保持现状（loopback 运维口） |
| POST `/api/stop-all` | Status UI session | System | 显式拆栈 | session_id / ownership | ✅ 充分 | 仍是唯一「停业务」入口 |
| 子进程 Setsid | 内核 | 进程 | 隔离 | n/a | ✅ | — |

## 安全审计

- [x] 无新 IDOR URL
- [x] 无跨租户数据
- [x] 无密钥入日志（只记录 pid/pgid/sid/trigger）
- [x] `/api/shutdown-self` 不扩大暴露面
- [x] env 拆栈开关不可被远程 HTTP 设置

## 权限测试

不新增 RBAC 测例。进程策略测例见测试意图 T1–T5。
