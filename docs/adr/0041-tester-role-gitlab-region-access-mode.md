# ADR-0041: 测试角色与 GitLab 区域开发/发布模式

- **Status:** accepted
- **Date:** 2026-08-23
- **Author:** cursor
- **Deciders:** goal-mode auto

---

## Context

需要一批「测试」账号在未对全体租户开放的 GitLab 镜像区域上联调。测试账号的产品权限应与租户相同，但不能获得系统管理平台角色。区域需要显式的开发/发布模式，开发模式仅测试账号可使用。taskBill 不得直连 `auth_user`。

## Decision

We will:

1. 在 `auth_user` 增加 `is_tester` 标志；勾选时强制 `is_tenant=1`。不把 tester 写入 `X-User-Roles`。
2. 网关 forward-auth 注入 `X-User-Is-Tester: 1`。
3. 在 `billing_gitlab_region` 增加 `access_mode`（`release` 默认 / `development`）。
4. 租户侧区域目录与购买/连接/Navbar 摘要按 `CanUseGitlabRegion(access_mode, is_tester)` 门禁；系统管理配置页不过滤。
5. 成功写路径发布 `UserTesterFlagChanged` 与 `GitlabRegionAccessModeChanged`。

## Alternatives Considered

### Alternative 1: 平台 RBAC 角色 tester

- **Pros:** 复用角色表与 X-User-Roles
- **Cons:** 平台角色会进入系统管理判定；与「等同租户」语义冲突
- **Why rejected:** 安全边界错误

### Alternative 2: 仅前端隐藏开发区域

- **Pros:** 改动小
- **Cons:** API 可绕过
- **Why rejected:** 违反 Enforce 同源

### Alternative 3: taskBill 回调 taskAuth 查 is_tester

- **Pros:** 不增网关头
- **Cons:** 每个区域列表多一次跨服务调用
- **Why rejected:** 已有 forward-auth 注入模式（邮箱、角色）

## Consequences

### Positive

- 生产区域默认发布，存量租户无感。
- 测试账号与租户共用控制台，无需第二套产品权限。

### Negative / Trade-offs

- GitLab 实例 OIDC 直连本期不按区域拦截。
- 区域从发布改为开发后，已购非测试租户会失去 SaaS 侧入口。

### Mitigations

- 超管改模式前在 UI 标明「开发模式仅测试角色可用」。
- OIDC 直连拦截记 OPT。

## References

- 设计: `docs/superpowers/specs/2026-08-23-tester-role-gitlab-region-access-mode-design.md`
- 单库所有权: `.ai/01_project_constraints/19_single_service_data_ownership.md`
