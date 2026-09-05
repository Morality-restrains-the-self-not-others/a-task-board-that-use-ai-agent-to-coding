# NFR 澄清 — 创建项目 ssh:// Git URL

- **Date:** 2026-09-02

## 路径分片键审视

| 路径 | 分片 ID | 适配性 | 级别 | 动作 |
|------|---------|--------|------|------|
| `/tenant/{tid}/create-project/` | `tenantId` | 租户壳 | L0 | 无新路由 |
| POST validate-git-repos `tenant_id/{tid}/` | `tenant_id` | 已有 | L1 | 不改路径 |
| 创建/PATCH 项目 | `tenant_id` + project | 已有 | L1 | 不改 |

全部路径已带租户键；本增量不引入无键列表。

## 幂等性审视

| 路径 | 副作用 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|----------|--------------|--------|----------|
| FE `isValidGitRepoUrl` | 无 | 输入/blur | — | — | L0 |
| validate-git-repos | 无写（探测） | blur/重试 | URL 行 | 现网 | L0 |
| 创建项目 | 写项目 | 双击 | 前端 clickGuard + 服务端 | 现网 | 不改资金路径 |

资金/配额：不涉及。

## 其它

- **安全 L2:** 只扩 `ssh:` scheme；规范化后走 HTTPS provider API。
- **可观测性:** 格式失败仍返回既有 message；不记录 token。
