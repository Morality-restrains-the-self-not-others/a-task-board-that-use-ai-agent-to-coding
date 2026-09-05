# 权限分析：relay 启动所选镜像

- 日期：2026-07-10
- 设计：`docs/superpowers/specs/2026-07-10-relay-start-selected-image-design.md`

## 变更面

| 接口 | 变更 | 鉴权 |
|------|------|------|
| `POST .../relay-to-trae/start/` | body 增 `installed_image_id` | 既有会话 + 租户/工作区/任务 scope（Gateway authorizeContainerRequest） |
| `GET /api/internal/mock-run/resolve-image` | 复用，无新公开 API | 内部 secret |
| `go_relayToTrae POST /v1/start` | 增 `image` | 既有 X-Relay-To-Trae-Secret |

## 结论

- 无新角色；无跨租户读镜像（resolve 按 tenant_id + installed_image_id）。
- 用户只能启动本租户已安装镜像。
- **无需新增权限模型**；审计沿用 RELAY_START_* 事件，日志应记录 `installed_image_id` / `image`（脱敏无密钥）。
