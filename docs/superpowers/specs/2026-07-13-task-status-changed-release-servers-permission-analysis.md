# 角色权限分析：任务状态变更事件与终态释放服务器

- 日期：2026-07-13
- 设计文档：`docs/superpowers/specs/2026-07-13-task-status-changed-release-servers-design.md`
- 状态：已完成（goal-mode 自动推进）

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| PATCH todos（既有） | workspace 成员（可变任务者） | Task / Workspace | write | `hasWorkspaceAccess` + `canMutateTaskFromRequest` | ✅ 充分 | 状态变更事件仅在鉴权通过且写库成功后发布 |
| Kafka `TASK_STATUS_CHANGED` 发布 | taskTaskService（系统） | Task | emit | 继承 PATCH 鉴权结果 | ✅ | payload 必须带 tenant/workspace/task 边界 |
| Consumer 释放编排 | taskEvents（服务账号） | CloudServerConfig / Task | write（副作用） | 服务间凭据；按 payload 边界查配置 | ✅ | 禁止仅用 task_id 跨租户查；校验 tenant/company 一致 |
| 发 `CLOUD_SERVER_STOPPED` | taskEvents | ECS Instance | delete | 既有 cloudserverstopped 链路 | ✅ | stop_reason 区分人工/终态自动 |
| HTTP relay/mock stop | taskEvents → ContainerGateway | Runtime | stop | 内部调用 / 服务凭据 | ⚠️ 确认 | 使用 internal 或服务间鉴权，勿伪造用户 Cookie |

## 结论

- **不新增公网 API**，无新角色。
- 用户侧权限完全复用 todos PATCH。
- Consumer 为异步系统副作用：须严格按事件 payload 中的租户/工作区/任务边界操作，防止 IDOR 式跨任务释放。
- 建议在 consumer 加载 CloudServerConfig 时校验 `tenant_id`/`company_id` 与事件一致。
