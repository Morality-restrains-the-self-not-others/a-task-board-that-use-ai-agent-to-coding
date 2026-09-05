# 角色权限分析 — TASK_FEATURE_PARAMS_SCOPE

- **日期**: 2026-07-09
- **设计文档**: `docs/superpowers/specs/2026-07-09-feature-params-scope-env-design.md`

## 结论

**无新增端点、无新增角色、无权限模型变更。** 仅在既有授权路径的响应/env 载荷中增加只读字段。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET/POST 公司 feature-params `env_preview` | tenant 管理员/有设置权限成员 | Tenant | read/write（既有） | 既有 tenant 权限 | ✅ 充分 | 无 |
| GET/POST 工作空间 feature-params | workspace 有权限成员 | Workspace | read/write（既有） | 既有 workspace 权限 | ✅ 充分 | 无 |
| 个人 feature-params-configs | 配置 owner | User | read/write（既有） | 既有 owner 校验 | ✅ 充分 | 无 |
| POST feature-params-env（容器） | 容器 token | Task | read | 既有 container token | ✅ 充分 | scope 来自任务绑定解析，非客户端传入 |

## 风险

- **IDOR**：无。scope 由服务端根据页面上下文或任务绑定计算，客户端不可指定覆盖。
- **信息泄露**：取值仅为枚举字符串，无敏感内容。
