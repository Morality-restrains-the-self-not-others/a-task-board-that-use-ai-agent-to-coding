# 测试意图：导航栏「代码仓库」按租户仓库资源跳转

## 测试目标

证明「代码仓库」入口按 `resources[]` 与拉取状态决定：就绪且空列表 → `/pricing/`；失败/未就绪 → 当前页；有资源 → 直链或下拉。不再用非 VIP1 作为跳转门禁。

## 测试分层

| 层 | 文件 | 覆盖 |
|----|------|------|
| 单元（Go） | `taskBill/src/gitlab_resources_test.go` | `listTenantGitlabNavTargets` 多区域 / 空列表 |
| HTTP（Go） | `taskBill/src/gitlab_convention_path_http_test.go` | GET 无 region 含 `resources` 且含赠送区域 web_url |
| 组件（Vue） | `taskFE/app/src/components/Navbar.ui.test.js` | 无资源 ready→pricing；error/unknown→当前页；单资源外链；多资源下拉 |
| 逻辑（Vue） | `taskFE/app/src/components/Navbar.logic.membershipTier.test.js` | 拉取 gitlab-resources 并透传 gitResources / gitResourcesStatus |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 租户无资源且 status=ready | href=`/pricing/`（保留 accessCode） |
| T1b | status=error 或 unknown | href=当前 `fullPath`，不含 `/pricing/` |
| T2 | 赠送 1 区 `tencent-sh-1` 且 membership=normal | 直链该区 `gitlab_web_url`，非 `/pricing/` |
| T3 | 两区均有 disk_gb>0 | 下拉两项，href 分别为两区 web_url |
| T4 | membership=vip1 无资源 ready | 显示 VIP1 角标，href=`/pricing/` |
| T5 | 未登录 | 不渲染 `nav-git-service` |

## 数据与环境

- Go：`setupMySQLTestDB` + `adminGrantResources` 指定 region。
- Vue：`vi.stubEnv` 不再作为跳转 SSOT；由 `gitResources` prop 驱动。

## 通过标准

上述测例全绿；既有 convention GET 无 region 仍返回 latest view 字段（向后兼容）。
