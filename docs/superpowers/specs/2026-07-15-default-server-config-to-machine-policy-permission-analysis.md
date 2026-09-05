# 权限分析：默认服务器启动配置迁入机器节点策略

- 日期：2026-07-15
- 依据设计：`2026-07-15-default-server-config-to-machine-policy-design.md`

## 结论

**无新增权限点、无新增 API。** 入口迁移不改变鉴权边界。

| 能力 | API | 既有角色边界 | 变更 |
|------|-----|--------------|------|
| 读/写默认启动配置 | `GET/POST .../cloud/server-config-default/` | 租户成员 + 云平台授权权限（既有） | 仅 UI 入口位置 |
| 读 CPA | `GET .../cloud/cloud-platform-authorizations/` | 同机器策略模态 | 无 |
| 机器策略 | `GET/PUT .../workspace-machine-policy/` | 工作空间管理权限（既有） | 无 |

## 风险

- 能开「机器节点」但不能写默认配置的用户：配置模态应表面有错误信息（既有 API 行为），不另开权限矩阵。

## 审计清单

- [x] 无 Django/Flask 新路由
- [x] 无 Go 新服务
- [x] Swagger 无需变更
