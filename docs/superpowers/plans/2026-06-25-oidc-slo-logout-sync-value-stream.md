# Value Stream: OIDC SLO 登出同步

> Derived from design: `docs/specs/oidc-slo-logout-sync-设计文档.md`

## Value Summary

用户在主应用退出登录时，通过 OIDC SSO 登录的 GitLab 代码仓库同步退出，实现真正的单点登出（Single Logout）。

## Related Value Streams

- **taskauth-oidc-issuer-docker-reachability**: dependency — SLO 依赖 OIDC issuer 在 Docker 网络中可达
- **oidc-ssl-protocol-fix**: dependency — SLO 依赖 OIDC SSL fix 使 GitLab ↔ taskAuth 通信正常
- **oidc-sso-404-fix**: extension — SLO 在 SSO 登录修复基础上补全登出链路，同属 OIDC 体系完善

## End-to-End Flow

```
[用户点击退出]
  → [主应用销毁 Django session + 清除 cookies]
  → [taskAuth 销毁 token + 清除 userId cookie]
  → [浏览器重定向至 GitLab sign_out]
  → [GitLab 销毁自身 session]
  → [重定向回主应用登录页]
  → [用户三方全部登出]
```

## Value Increments

### Increment 1: taskAuth EndSession 端点（Thin Slice）
**Value to user:** taskAuth 具备通知 RP 登出的能力（基础设施）
**Scope:**
- 新增 `GET /api/oidc/endsession` 处理函数
- OIDC Discovery 添加 `end_session_endpoint`
- 清除 userId cookie + 销毁 customtoken
- 302 重定向至 GitLab sign_out
**Depends on:** taskauth-oidc-issuer-docker-reachability, oidc-ssl-protocol-fix, oidc-sso-404-fix

### Increment 2: 主应用服务端登出
**Value to user:** 主应用登出真正销毁服务端 session（修复空函数体 bug）
**Scope:**
- `logout()` 视图调用 `auth_logout()` 销毁 Django session
- `UserViewSet.logout()` API 返回 SLO 重定向 URL
- 新增 `GIT_SERVICE_PUBLIC_BASE` Django 配置
**Depends on:** Increment 1

### Increment 3: 前端 SLO 重定向
**Value to user:** 用户点击退出后自动完成三方登出链（端到端可用）
**Scope:**
- `Navbar.logic.vue` 登出成功后执行 SLO 重定向
- 跳转 GitLab sign_out → 回到主应用登录页
**Depends on:** Increment 2

### Increment 4: Playwright E2E 测试
**Value to user:** 自动化回归保护，确保 SLO 行为不被破坏
**Scope:**
- Bug 复现测试（基线）
- 完整 SLO 登出同步测试
**Depends on:** Increment 3
