# 设计文档：Git 网站授权页 connection GET 500 修复

**日期：** 2026-05-27  
**状态：** 已实施  
**页面：** `http://localhost:4000/user/827923618451263488/profile/git-site-oauth/`

---

## 现象

用户已登录并访问本人 Git 网站授权设置页时，前端调用：

```
GET /api/user/{user_id}/accounts/github/app/connection/
```

返回 **500 Internal Server Error**，页面显示「暂时无法获取绑定状态」。

浏览器 Network 响应体（Django debug）：

```
TypeError: get() got an unexpected keyword argument 'user_id'
Raised during: accounts.github_app_views.GithubAppConnectionView
```

同页 `GET .../accounts/users/me/` 正常 200。

---

## 根因

`saas_project/urls.py` 将 OAuth 连接状态路由迁移为 **user-scoped** 形式：

```python
path(
    'api/user/<str:user_id>/accounts/github/app/connection/',
    GithubAppConnectionView.as_view(),
    ...
)
```

Django 会将 URL 捕获组 `user_id` 作为关键字参数传入视图方法。

| 方法 | 签名 | 结果 |
|------|------|------|
| `GithubAppConnectionView.delete` | `delete(self, request, user_id=None)` | ✅ 正常 |
| `GitlabAppConnectionView.get` | `get(self, request, user_id=None)` | ✅ 正常（子类 override） |
| **`GithubAppConnectionView.get`** | **`get(self, request)`** | ❌ 缺少 `user_id`，500 |

这是路由迁移时的**签名遗漏**，非 gitOauth 或前端问题。

---

## 方案

### 1. 修复视图签名（必须）

在 `GithubAppConnectionView.get` 增加与 `delete` / `GitlabAppConnectionView.get` 一致的参数：

```python
def get(self, request: Request, user_id: str | None = None):
```

业务逻辑仍使用 `request.user.pk`（与 delete 一致）；`user_id` 仅用于吸收 URL kwargs，避免 TypeError。

### 2. 回归测试（必须）

新增 pytest：`GET /api/user/{user_id}/accounts/github/app/connection/` 在 Token 认证下返回 200（mock gitOauth summary），防止再次遗漏签名。

### 3. 手工验收

- 刷新 `git-site-oauth` 页，connection 请求 200
- GitHub / GitLab tab 切换均可加载绑定状态

---

## 价值流影响

| 流 | 影响 |
|----|------|
| `gitoauth-binding-state-persistence` | 读连接状态步骤恢复可用 |
| `oauth-callback-error-toast` | 设置页内联错误展示依赖 connection GET |
| `task-detail-oauth-binding-guidance` | 间接依赖同一 connection API |

不涉及新字段、新路由或 gitOauth 服务变更。

---

## 领域概念（轻量）

- **Bounded Context：** 用户账号 / Git OAuth 绑定（accounts + gitOauth 桥接）
- **实体：** User、GitHubAppUserCredential（gitOauth 侧）
- **领域服务：** `fetch_gitoauth_provider_credential_summary_for_user`
- **值对象：** ConnectionSummary（connected、github_login、scope、connections[]）

---

## 非目标

- 不修改 URL 结构或前端 `apiConnectionUrl` 构造
- 不重构 gitOauth 摘要逻辑
- 不在此变更中增加 URL `user_id` 与 session 不一致时的额外校验（现有 `IsAuthenticated` + 前端 `isOwnProfile` 已覆盖）

---

## 风险

- **低**：单行签名 + 测试，与已有 delete/gitlab-get 模式一致
- Django 开发服务器需 reload 后生效（或 runAll 重启 saas-backend）
