# 权限分析 — zTree 合并到目标分支

## 角色与边界

| 角色 | 能力 |
|------|------|
| 任务成员（已登录 + 租户/工作区范围） | 可调用 `container-layer-git-merge`（与 commit/push 同授权链） |
| 未授权 / 跨租户 | authorizeContainerRequest / Django 会话拒绝 |
| 容器 ACCESS_TOKEN | 网关持 token 调用 onlineServiceJS |

## 新增接口权限

- 浏览器 → taskGateway → taskContainerGateway（session）→ onlineServiceJS（X-Access-Token）
- 无新权限模型；复用容器计算 outbound 授权
- 无支付/KYC 面

## 审计

- 容器：结构化 log `[LayerGitMerge]`
- SaaS/网关：既有 outbound / reqLogs 路径
