# NFR Clarification — navbar-git-service-region-nav

默认 L2；本增量无资金写路径。

| 类别 | 级别 | 要求 |
|------|------|------|
| 正确性 | L2 | 赠送与购买同等出现在 `resources[]` |
| 安全 | L2 | 不返回 `admin_private_token`；外链 noopener |
| 可观测 | L2 | GET 已有 `gitlab_resources_get_ok`；列表条数可附 `resource_count` |
| 性能 | L2 | 单次 JOIN 查询，禁止 N+1 打区域 |
| 可用性 | L2 | 列表失败 fail-open：当无资源处理（留当前页），不阻断导航栏 |

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| GET `/api/billing/gitlab-resources/tenant_id/{tid}/` | `tenant_id` | 适配（租户账本伸缩要素） | 查询必须带 tenant_id |
| GET `/api/tenant/{tid}/billing/gitlab-resources/` | `tenant_id` | 同上 | 同上 |
| 前端 `/tenant/:tenant/settings/gitlab-connection/` | `tenant` | 适配 | 无改 |
| Navbar 全站（无 tenant 段时） | 用 `currentTenant` | 无租户则不请求 | L0：无租户无仓库数据 |

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 幂等键 | 判定 | 等级 | 重放语义 / 动作 |
|------|--------|------------|--------|------|------|-----------------|
| GET gitlab-resources | 无（只读；regionResourceView 内 best-effort 刷新用量不改变配额所有权） | — | — | 只读 | L0 | 纯查询；Navbar 只读 `resources[]` |
| 点击导航 `<a>` | 无平台写 | — | — | 只读 | L0 | 浏览器导航，无副作用 |
