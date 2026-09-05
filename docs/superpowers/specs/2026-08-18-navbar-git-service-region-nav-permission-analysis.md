# 角色权限分析 — 导航栏代码仓库区域跳转

无新写接口、无新角色。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/billing/gitlab-resources/tenant_id/{tid}/` | 已登录租户成员（网关 JWT） | Tenant | read | 网关注入用户；path 含 tenant_id | 与现网 GET 同级；未新增 IDOR 面 | 保持既有网关鉴权，不在本次加 billing:manage |
| Navbar 下拉外链 | 已登录且 `isUserAuthenticated` | Tenant 资源跳转 | 导航 | 仅登录可见 | ✅ | 未登录不渲染 |
| 菜单项 gitlab_web_url | 浏览器打开公网 GitLab | 外部系统 | 只读跳转 | GitLab 自身 SSO | ✅ | 不把 PAT 下发前端 |

不引入新权限码。系统管理入口与租户侧栏 GitLab 设置权限不变。
