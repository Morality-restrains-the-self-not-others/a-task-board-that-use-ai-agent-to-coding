# Application Integration v16 — 多入口同源 API

```mermaid
flowchart LR
  Browser[Browser] -->|HTTPS/HTTP 任意入口| Edge[Edge nginx]
  Edge -->|"/"| Vite[Vite Vue :4000]
  Edge -->|"/api/"| GW[task-gateway :18081]
  Vite -->|dev proxy /api| GW
  GW --> BE[saas-backend Django]
  Browser -.->|DEPRECATED Mixed Content| AbsAPI["http://IP:18081"]
```

## 变更摘要

- 🟢 Edge nginx：多域名/IP 入口统一 `/` → Vue、`/api/` → gateway
- 🟡 Vue：`API_BASE_URL` 默认同源相对路径
- 🟡 Vite：恢复 `/api` → gateway 代理
- 🔴 废弃浏览器默认绝对 `http://IP:18081` API 基址（HTTPS Mixed Content）
