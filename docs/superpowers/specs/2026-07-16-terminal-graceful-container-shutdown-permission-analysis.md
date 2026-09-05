# 权限分析：terminal-graceful-container-shutdown

- 日期：2026-07-16
- 设计：`docs/superpowers/specs/2026-07-16-terminal-graceful-container-shutdown-design.md`

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| PATCH todos（既有） | workspace member w/ task edit | Task | write | 既有鉴权 | ✅ | 不变 |
| POST `/api/task-lifecycle/shutdown`（容器内） | taskEvents / CGW 服务凭据或直连 | Container | write | 容器内网；无用户 cookie | ✅ | 不暴露公网无鉴权入口 |
| CGW L0 `container-task-lifecycle-shutdown` | 已登录用户 + task 权限 | Task | write | CGW 既有 session/scope | ✅ | 与其它 container-* 一致 |
| POST `…/request-machine-release/` | 容器 access_token | Task | write | validateContainerAccessToken + path match | ✅ | 强制三段 ID 与令牌一致；禁止跨任务 |
| SoleContainerGate 查 bindings | 服务内部 | Workspace+Instance | read | internal 服务凭据 | ✅ | 仅本 company/workspace |
| TASK_STATUS_CHANGED 消费 | taskEvents | System | side-effect | 事件总线 | ✅ | payload 须带 tenant/workspace/task |
| 超时硬释放 | taskEvents | Task cloud | write | 同 v27 | ✅ | 幂等 mark terminal_released |

## 结论

- **不新增角色**
- **无 IDOR**：inbound 令牌绑定 task；path 不匹配 → 403
- **无新公网无鉴权 API**
