# Navbar 无仓库跳价格页 — NFR 澄清

- **日期**: 2026-08-29
- **价值流**: `docs/superpowers/plans/2026-08-29-navbar-git-empty-to-pricing-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| 前端 GET 导航 `/pricing/`（及 `?accessCode=`） | 无 tenant | L0 | 公开营销页，全站一份；升级触发：价格页按租户定制目录 |
| 前端当前页 `fullPath`（fail-open） | 可能含 `tenantId` | L0 | 仅降级 href，不新增查询 |
| 既有 `GET /api/tenant/{tid}/billing/gitlab-resources/` | `tid` 租户键 | 适配 | 本迭代不改契约；tid 与计费资源表分片一致 |

可伸缩性：全部书面 L0（导航）+ 既有租户 GET。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 业务边界 | 幂等键 | 重放语义 |
|------|--------|----------|----------|--------|----------|
| 点击「代码仓库」→ `/pricing/` | 无（浏览器 GET） | 连点 | 页面导航 | L0 无副作用 | 重复打开价格页 |
| GET gitlab-resources | 无（只读） | 刷新 | 租户列表快照 | L0 | 重复拉取 |

资金/云资源：否（转化入口，不下单）。一致性 L0。

## 其它 NFR

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L1 | 真实 href；不拦截点击；accessCode 已在价格链使用 |
| 可用性 | L2 | API 失败 fail-open 留当前页，不误送价格页 |
| 可观测性 | L1 | fetch 失败 `console.warn`（无 token）；无新 MQ |
| 性能 | L0 | 无新请求 |

## 领域模型影响

无新聚合。沿用租户 GitLab 资源只读投影。
