# NFR 澄清: GitHub Connection GET user_id 签名修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-27-github-connection-get-user-id-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-27-github-connection-get-user-id-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | connection GET P95 ≤ 500ms（含 gitOauth 下游） |
| 可用性 | L2 | 修复后设置页 connection 成功率 100%（非 gitOauth 503） |
| 安全性 | L2 | IsAuthenticated + Token/Session，沿用现有策略 |
| 可维护性 | L2 | pytest 覆盖 user-scoped GET，防签名回归 |

## 质量场景

### QS-01: 设置页加载绑定状态

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 已登录用户 |
| 刺激 | GET `/api/user/{uid}/accounts/github/app/connection/` |
| 制品 | `GithubAppConnectionView.get` |
| 环境 | 正常，gitOauth 可用 |
| 响应 | HTTP 200，JSON 含 `connected` 字段 |
| 响应度量 | pytest 断言 status_code=200；浏览器 Network 无 500 |

## 领域模型影响

无新领域概念；修复限于 API 视图方法签名，不调整聚合边界。

## 跳过声明

- 可伸缩性、合规、容错专项：不适用，单行签名修复。
