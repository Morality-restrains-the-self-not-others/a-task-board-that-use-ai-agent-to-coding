# 多入口同源 API — 领域概念（轻量）

无新持久化聚合。运维配置值对象：

- **PublicEntryOrigin**: `scheme://host[:port]`，驱动 CSRF / ALLOWED_HOSTS / CORS / Vite allowedHosts
- **EdgeProxyRoute**: `/` → frontend，`/api/` → gateway

实现落在 conf + nginx + Vite，不新增 domain 代码目录。
