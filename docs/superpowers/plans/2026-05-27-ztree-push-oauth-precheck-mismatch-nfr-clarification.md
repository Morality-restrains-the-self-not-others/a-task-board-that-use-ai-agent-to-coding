# NFR 澄清: zTree 推送 OAuth 预检对齐

> 输入: design + value-stream for ztree-push-oauth-precheck-mismatch

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | push 预检移除后不增加额外 API；单次 push P95 ≤ 120s（沿用现有） |
| 安全性 | L3 | token 仍仅后端换票下发容器，前端不暴露 refresh_token |
| 数据一致性 | L2 | 推送凭证与克隆共用 gitOauth 换票路径 |
| 可维护性 | L2 | auth context 响应 additive 字段，向后兼容 |
| 可观测性 | L1 | 沿用现有 LayerGitPush 日志 |

## 质量场景

### QS-01: GitLab 已授权推送不被前端误拦
| 要素 | 内容 |
|------|------|
| 刺激源 | 任务详情用户 |
| 刺激 | zTree 点击推送，GitLab OAuth connected，GitHub connected=false |
| 制品 | `onLayerGraphLayerPush` |
| 响应 | 发起 POST container-layer-git-push |
| 响应度量 | 前端单元测试：无 alert，apiFetch 被调用 |

### QS-02: 缺 OAuth 时后端 409 可执行
| 要素 | 内容 |
|------|------|
| 刺激源 | API 客户端 |
| 刺激 | POST push，无 provider connected |
| 制品 | `forward_container_layer_git_push` |
| 响应 | 409 + provider 感知 detail |
| 响应度量 | pytest assert status 409 且 detail 不含误导性「仅 GitHub」 |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| L2 一致性 | PushOAuthReadiness 按任务仓库 providers 聚合 | 值对象 PushOAuthReadiness |
| L3 安全 | 换票仍在 gitOauth 防腐层 | 领域服务接口不变 |

## 权衡与边界

- 不做前端 push 前二次 auth-context 强预检（Increment 1 足够）
- 不修改 prefer_container_remote 策略
