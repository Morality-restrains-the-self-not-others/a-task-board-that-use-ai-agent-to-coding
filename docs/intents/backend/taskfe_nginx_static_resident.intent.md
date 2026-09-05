# Intent: 公网 taskFE 由 nginx 静态常驻

## 需求（2026-08-20）

runAll / 公网入口的 taskFE HTML 壳由 **Docker nginx 容器** 发布到宿主机 `0.0.0.0:4000`；Vite 只构建。构建只切 `public/html` symlink，**不 recreate 容器**，避免 APISIX `spa-catch-all` connection refused 502。

## 明确不改

| 项 | 原因 |
|---|---|
| APISIX `spa-catch-all` uris / upstream 端口 | 仍打 :4000 |
| Vue 路由 `/user/:id/profile/` 等 | 客户端路由，与 serving 无关 |
| SH 边缘 nginx | 只做 TLS/反代，不读 dist |
| `npm run dev` | 本机 HMR 仍用 Vite |

## 验收

见 [taskfe_nginx_static_resident.test-intent.md](./taskfe_nginx_static_resident.test-intent.md)。

## 业务意图 → 事件对照

**无对应事件**：静态托管，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 公网 SPA nginx 常驻 | — | — | — | — | 基础设施 serving，无领域状态变更 |
