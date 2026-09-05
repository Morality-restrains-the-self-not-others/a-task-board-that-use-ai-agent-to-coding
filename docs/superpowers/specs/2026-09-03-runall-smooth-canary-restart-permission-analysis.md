# 角色权限分析：金丝雀平滑重启

- **日期:** 2026-09-03
- **设计:** `docs/superpowers/specs/2026-09-03-runall-smooth-canary-restart-design.md`

## 结论

无新角色。9999 生命周期 API 仍仅本机运维面（session_id 所有权），不面向租户。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST /api/restart-all | 运维（持有 session_id） | System | 写（进程生命周期） | session_id 必填；bulk 互斥 | ✅ 充分 | 不向公网暴露；保持既有 |
| POST /api/precise-restart | 同上 | System | 写 | session_id + 登记文件 | ✅ 充分 | — |
| GET /api/restart-all/progress SSE | 同上 | System | 读 | 无租户数据 | ✅ 充分 | — |
| tracelog.ListenAndServe | 服务进程 | System | 内部 | 无用户输入 | ✅ | — |

无新 RBAC 权限点；无跨租户数据面。
