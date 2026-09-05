# Application Integration v15 — relay 启动所选镜像

```mermaid
flowchart LR
  User[用户选镜像并启动] --> Vue[Vue ServerConfig]
  Vue -->|POST start + installed_image_id| CGW[taskContainerGateway]
  CGW -->|token-init| CRED[taskCredentialService]
  CGW -->|resolve-image| Cloud[taskCloudService]
  Cloud -->|image ref| CGW
  CGW -->|POST /v1/start + image| Relay[go_relayToTrae]
  Relay -->|docker pull + run| Docker[Docker Engine]
```

## 变更摘要

- 🟢 Vue → CGW 启动体携带 `installed_image_id`
- 🟢 CGW → Cloud `resolve-image`（复用 mock-run 内部接口）
- 🟡 go_relayToTrae：有 `image` 时 docker pull/run，不再仅跑 host `run.sh`
