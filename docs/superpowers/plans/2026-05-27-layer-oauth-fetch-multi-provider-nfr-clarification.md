# NFR 澄清: 容器层 OAuth 拉取 AccessToken 多 Provider

> 输入: 设计文档 + 价值流 `2026-05-27-layer-oauth-fetch-multi-provider-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 单仓换 token P95 ≤ 2s（含 gitOauth 一次 RTT） |
| 安全性 | L3 | 容器 AccessToken 校验 + token 文件 0600 |
| 数据一致性 | L1 | 无跨仓事务；单仓写入独立 |
| 容错 | L2 | gitOauth 超时结构化错误；已有 retry 策略复用 |
| 可维护性 | L2 | API 向后兼容 github_auth_by_repo |
| 可伸缩性 | L0 | 层内仓数 ≤10，无专项优化 |

## 质量场景

### QS-01: GitLab local 拉 token
| 要素 | 内容 |
|------|------|
| 刺激源 | 容器 UI 用户 |
| 刺激 | 层内 origin=localhost:8012，点击拉取 |
| 制品 | oauth-fetch-token-files + layer-github-oauth-access-tokens |
| 响应 | 200 + `.task2app_access_token` 写入 |
| 响应度量 | 单测 + 集成测断言文件内容 |

### QS-02: 无效容器 token
| 要素 | 内容 |
|------|------|
| 等级 | L3 安全 |
| 响应 | 401，不泄露 gitOauth 细节 |

## 领域模型影响

| NFR 决策 | 模型影响 |
|----------|---------|
| L2 性能 | 按 (user_id, provider_key) 缓存 token，同请求内 dedupe |
| L3 安全 | ContainerTokenContext 不变；RepoMatchKey VO 校验格式 |

## 权衡与边界

- 不做 Bitbucket；仅 gitOauth 已配置 provider
- 不重命名 TaskGithubRepoOauthBinding
