# DDD：容器执行热路径零 Django

- **日期**: 2026-07-10

## 限界上下文

| 上下文 | 职责 | 服务 |
|--------|------|------|
| Identity & Access | 解析会话/Token、租户成员 | taskAuth |
| Cloud Runtime Config | server_url / business endpoint、container-target | taskCloudService |
| Container Credential | access_token SSOT | taskCredentialService |
| Container Gateway | L0 编排与出站转发 | taskContainerGateway |
| Container Runtime | job 执行 | onlineServiceJS |

## 应用服务（tcg）

```text
HandleContainerCompute(req):
  session = Auth.ValidateSession(cookie, authz, tenant, path)
  if !session.ok → 401/403
  if !relayPath(path):
    cfg = Cloud.Lookup(tenant, workspace, task) // 可选，或并入 Resolve
  target = Cloud.ResolveContainerTarget(scope, overrideURL)
  if target missing → 404/409
  return Forward(onlineServiceJS, target)
```

## 防腐层

- 废弃：tcg → Django `validate-session` / `resolve-container-target`（热路径）
- 新增/启用：tcg → taskAuth validate；tcg → Cloud container-target

## 领域事件

本 Phase 不新增事件。
