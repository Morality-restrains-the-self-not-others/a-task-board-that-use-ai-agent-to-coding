# v13 application-integration — Container Exec Hot-Path Zero-Django

```mermaid
flowchart LR
  Vue["Vue Frontend"] -->|container-layer-command / job-*| APISIX["task-gateway APISIX"]
  APISIX --> CGW["taskContainerGateway :8014"]
  CGW -->|auth_validate| AUTH["taskAuth :8003"]
  CGW -->|cloud_resolve container-target| TCS["taskCloudService :8018"]
  TCS -->|token by-scope| CRED["taskCredentialService :8015"]
  CGW -->|POST /api/jobs| OSJS["onlineServiceJS"]

  CGW -.->|DEPRECATED hot-path| DJ["Django validate/resolve"]
```

## 架构变迁

```mermaid
flowchart LR
  P12["Plateau v12"] --> Gap["Gap: tcg 每请求 Django internal"]
  WP["WP-v13-container-exec-zero-django"] --> Gap
  WP --> P13["Plateau v13: Auth+Cloud 热路径"]
```

- Plateau v12 → Gap: 容器执行仍经 Django `validate-session` / `resolve-container-target`（saas-backend 日志可见；Django 宕则 502）
- WorkPackage: tcg 改接 taskAuth + taskCloudService
- Plateau v13: 执行热路径零 Django
