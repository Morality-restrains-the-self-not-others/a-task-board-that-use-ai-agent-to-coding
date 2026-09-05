# 设计：测试角色 + GitLab 镜像区域开发/发布模式

- **日期**: 2026-08-23
- **页面**: `/system-admin/users/`、`/system-admin/gitlab-resources`
- **状态**: accepted（goal-mode 自动采用）
- **架构变更**: 是（安全策略 + 跨服务鉴权头 + 新列 + 领域事件）
- **ADR**: [ADR-0041](../../adr/0041-tester-role-gitlab-region-access-mode.md)

## Context

系统管理员用户列表角色筛选项现为：超管 / 员工 / 租户 / 用户。需要增加「测试」类别：产品权限与租户相同（可建公司、用租户控制台、购买/连接 GitLab），用于在未对全体租户开放的 GitLab 镜像区域上做联调。

GitLab 区域管理页（镜像区域卡片）需要开发模式 / 发布模式。开发模式下该区域**仅测试角色账号可以使用**（目录可见、下单、OAuth 连接、Navbar「代码仓库」入口）。存量区域默认发布模式，避免误锁生产。

## 当前架构理解

- 账号类别由 `auth_user` 标志位表达：`is_superuser` / `is_staff` / `is_tenant`（非平台 RBAC 角色）。
- 平台员工走 `X-User-Roles`（`super_admin` / `employee`）；租户不是平台角色。
- GitLab 区域元数据在 `taskBill.billing_gitlab_region`；租户目录 `GET` 区域列表、购买、配额均在 taskBill。
- 网关 forward-auth 由 taskAuth 注入 `X-User-Id` / `X-User-Roles` 等；下游不得直查 `auth_user`。

## Decision

### 1. 测试角色 = 租户能力 + `is_tester` 标志

- 新增 `auth_user.is_tester TINYINT(1) NOT NULL DEFAULT 0`（`dataMigrate/taskAuth/039_user_is_tester.sql`）。
- **不是**平台 RBAC 角色，禁止写入 `X-User-Roles`（避免被 `IsPlatformStaff` 当成系统管理身份）。
- 勾选测试时后端强制 `is_tenant=1`（测试等同租户）；取消测试不自动取消租户。
- 列表展示优先级：超管 > 员工 > **测试** > 租户 > 用户。
- 过滤器 `role=tester` → `is_tester=1`；`role=tenant` → `is_tenant=1 AND is_tester=0 AND 非超管/员工`。
- `/me/`、系统管理用户 JSON 增加 `is_tester`。
- 网关注入 **`X-User-Is-Tester: 1`**（仅测试账号）；forward-auth 缓存同步该标志，PATCH 后失效缓存。

### 2. 区域访问模式

- `billing_gitlab_region.access_mode VARCHAR(16) NOT NULL DEFAULT 'release'`（`dataMigrate/taskBill/065_gitlab_region_access_mode.sql`）。
- 枚举：`release` | `development`。非法值拒绝。
- 系统管理列表/创建/编辑展示并保存该字段。
- 租户侧目录：`listGitlabRegions` 对非测试调用方过滤掉 `development`；测试账号可见全部启用区域。
- 写路径（购买、开通、GitLab OAuth 绑区、Navbar 已购区域摘要）同样门禁：开发模式且非测试 → **403**，文案「该区域处于开发模式，仅测试角色账号可使用」，带 `trace_id`。
- 系统管理配置页不过滤（超管必须能改模式）。

### 3. 跨服务

- taskBill / taskGitOauth 只读 `X-User-Is-Tester`，禁止直连 auth 库。
- 领域规则集中：`CanUseGitlabRegion(accessMode, isTester)` — `release` 一律允许；`development` 仅 `isTester`。

### 4. 事件

| 意图 | 事件 | Topic | 消费者 |
|------|------|-------|--------|
| 设置/取消测试角色 | UserTesterFlagChanged | user-tester-flag-changed | 无自动消费者（审计）；handler 内清 forward-auth 缓存 |
| 更改区域模式 | GitlabRegionAccessModeChanged | gitlab-region-access-mode-changed | 无自动消费者（审计） |

纯 GET 目录过滤无事件。

### 5. 前端

- 用户新增/编辑：增加「测试」复选框（`data-alias="SystemAdminUsersFilterRole"` 筛选项增加「测试」）。
- 区域卡片：显示模式徽章；编辑/新建可选「开发模式 / 发布模式」。
- 副作用按钮（保存配置）已有保存中 disabled；模式切换走既有 PUT，沿用同步门闩。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 把 tester 放进 `X-User-Roles` | 易被当成平台员工，误入系统管理 |
| 独立 RBAC 角色 `tester` | 与「等同租户」冲突；租户能力来自 `is_tenant` 而非平台角色 |
| 用 `is_active=0` 模拟开发区 | 停用会连超管配额视图一起隐藏，无法定向开放 |
| 仅前端隐藏开发区 | API 可绕过；必须 BE Enforce |
| 开发区对超管也开放「使用」 | 需求写明仅测试角色；超管只管理配置 |

## Consequences

- 存量区域默认 `release`，生产租户无感。
- 非测试租户若历史上已购某区、之后该区改为 development：目录与 Navbar 隐藏该区，购买/连接 403。
- GitLab 实例自身 SSO（OIDC）本期不按区域拦截直连登录（映射 client→region 未就绪）；SaaS 侧选择/购买/连接已封锁。记 OPT。

## 契约

用户 PATCH 增可选字段 `is_tester: bool`。

区域 JSON 增：

```json
{ "access_mode": "release" }
```

区域列表（租户）：测试账号含 development；其他账号仅 release。

系统管理区域列表始终含 `access_mode`。

## 🕸️ Code Review Graph 分析

- CRG `update --brief`：增量 6 files / 0 nodes（本增量开始前工作区与图一致）。
- 影响符号：`listGitlabRegions`、`handleGitlabRegionsList`、`patchUserAsAdmin`、`parseSystemAdminUserListFilters`、`writeForwardAuthHeaders`、`SystemAdminUsersFilters.roleOptions`、`handleSystemAdminUpdateRegionMetadata`。
- CodeGraph CLI 可用；无 MCP `codegraph_explore`。

## Python 新增接口清单

无。全部落现有 Go 服务（taskAuth / taskBill）扩展字段与门禁。

## 架构交付物

版本 **v105**（基于 v104 current）：

- `docs/architecture/v105-application-integration-20260823-2056-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
- `docs/architecture/v105-enterprise-landscape-20260823-2056-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
