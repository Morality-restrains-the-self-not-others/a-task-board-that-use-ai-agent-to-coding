# 任务详情 container-auto-run-steps 409「缺少容器 access_token」

- **日期**: 2026-07-19
- **症状**: `GET …/cloud/compute/container-auto-run-steps/` → **409**，body ≈ `{"detail": "缺少容器 access_token"}`（content-length 39）；traceId 如 `a727a391-2fa7-4f60-99b1-3248bac36328`、`5ab6184d-3481-415a-ad77-5963dad69b92`。
- **链路**: APISIX → taskCloudService 代理 → taskContainerGateway → Cloud `container-target` → credential `GET /v1/token/by-scope`。
- **根因（两阶段）**:
  1. **exchange 清空 access**：`exchange-refresh` 成功后 `UpdateAccessToken("", "")`；旧 by-scope 用 `IsExpired`（expires 空 → 未过期）→ **200 返回空 access_token** → Cloud `fetchTokenByScope` 空串 → `resolveContainerTarget` 有 baseURL 无 token → 409。
  2. **access 非空但已过期（本例）**：EnsureAccess 曾对过期非空 access 返回 `TOKEN_EXPIRED` → by-scope **404** → `fetchTokenByScope` 空串 → 同样 409。容器 `auth.mjs` 按 `process.env.ACCESS_TOKEN` **字符串比对**、不校验平台 `expires_at`，单方 Refresh 会换新 token 导致全线 401；正确做法是**原样返回过期非空 access**。
- **修复**:
  - `TokenService.EnsureAccessByScope`：
    - access 非空（含已过期）→ 原样返回，不单方 `RefreshAccess`；
    - access 空且有 refresh → 自动 `RefreshAccess`；
    - access 空且无 refresh → `TOKEN_NOT_FOUND`；
  - `handleGetTokenByScope` 走 Ensure；
  - 前端 409 且有 installed 缓存时静默回退（不刷 liveError）。
- **复验**（task_13571260012264162867）:
  - by-scope：200 + 过期 `expires_at` 仍带原 access；
  - container-target：200；
  - 公网 `container-auto-run-steps`：200 + markdown。
- **预防**:
  - SaaS→容器凭据解析不得依赖「by-scope 在 exchange 后仍持有非空 access」；空 access + 有 refresh 必须自愈。
  - 不得因平台 TTL 过期而拒绝转发仍可能与容器 env 一致的 access。
  - 长跑 **onlineServiceJS** 须主动续签：`proactiveAccessRefresh.mjs`（对齐 go_relay；OPT-036/046 已完成并推镜像滚动）。
  - `container-target` 409 须带 `error_code` / `credential_status`（OPT-045），勿只返回笼统「缺少容器 access_token」。
