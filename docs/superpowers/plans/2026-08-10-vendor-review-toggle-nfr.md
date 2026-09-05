# Step 5 — NFR（L2）

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L2 | admin PATCH 需平台角色/staff；公开 GET 仅布尔 |
| 可用性 | L2 | 设置读失败默认审核开启（保守） |
| 性能 | L2 | 单行表；可进程内短 TTL 缓存（可选） |
| 可观测 | L2 | MarketplaceSettingsUpdated / BridgeAutoProvision 日志 |
| 兼容 | L2 | vendor-status 新字段可选；默认审核 on |
