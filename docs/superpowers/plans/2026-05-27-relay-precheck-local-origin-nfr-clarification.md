# NFR 澄清: relay 直启预检内网 origin

> 输入: 设计文档 + 价值流 `2026-05-27-relay-precheck-local-origin-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 预检 P95 ≤ 800ms |
| 容错 | L2 | 网关不可达返回明确 5xx 文案，不误导 OAuth |
| 可观测性 | L2 | 保留 trace_id；502 含 origin 提示 |
| 可维护性 | L2 | pytest 锁定 internal origin URL |
| 安全性 | L2 | 预检仍用 container token，不走公网 |

## 质量场景

### QS-01: 本地预检走内网
| 要素 | 内容 |
|------|------|
| 类别 | 容错 |
| 等级 | L2 |
| 刺激 | vue.apiBaseUrl 为公网，用户直启 |
| 制品 | repo-credentials-precheck |
| 响应 | POST 目标为 internalApiBase |
| 响应度量 | pytest 断言 URL host=127.0.0.1:8001 |

### QS-02: 502 不误导 OAuth
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激 | 预检返回 502 |
| 制品 | ServerConfig 启动面板 |
| 响应 | 不展示 OAuth 引导卡片 |
| 响应度量 | 文案不含「完成 OAuth 绑定」 |

## 领域模型影响

复用现有 `TaskRepoCloneCredentialsGuardService`；预检 origin 选择为基础设施层 concern，不新增聚合。

## 权衡

- 容器 runtime 仍可使用公网 TASK_API_ENDPOINT_ORIGIN
- 预检与 runtime origin 分离，避免本地 token 远端校验
