# 角色权限分析：任务终态硬释放 + 镜像容器迁移

- 日期：2026-07-15
- 设计文档：`docs/superpowers/specs/2026-07-15-terminal-hard-release-container-migrate-design.md`
- 状态：已完成（goal-mode 自动推进）

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| PATCH todos（既有） | workspace 可变任务成员 | Task | write | hasWorkspaceAccess + canMutateTask | ✅ | 事件仅写库成功后发 |
| GET internal instance-bindings | taskEvents 服务账号 | Workspace / CSC | read | 服务间凭据 | ⚠️ 须加 | 强制 company_id+workspace_id 与 path/query 一致；禁止仅 instance_id |
| POST migrate-container-off-instance | taskEvents | Task CSC + Runtime | write | 服务间凭据 | ⚠️ 须加 | body 校验 tenant/workspace/owner_task；forbid 跨租户 start-vm |
| POST mark-terminal-released | taskEvents | Task CSC | write | 服务间凭据 | ⚠️ 须加 | 同边界校验 |
| Handler migrate→stop 编排 | taskEvents | Cloud / Container | side-effect | payload 边界 | ✅ | 加载 bindings 后二次校验 company/workspace |
| idle reuse 解绑 source | taskCloudService | CSC | write | start-vm 用户鉴权 | ✅ | 解绑不得越权清其他 workspace |

## 结论

- **不新增公网 API / 角色**。
- 用户权限完全复用进度 PATCH。
- 新增 internal API 必须服务凭据 + 租户/工作区边界，防 IDOR 式跨任务迁机或误释放。
- migrate 触发的 start-vm 走服务凭据内部路径时，仍须绑定 owner_task 的 workspace。
