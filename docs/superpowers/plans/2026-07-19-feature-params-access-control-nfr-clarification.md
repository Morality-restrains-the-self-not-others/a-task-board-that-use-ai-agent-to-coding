# Feature-Params 访问控制 — NFR

| 类别 | 级别 | 说明 |
|------|------|------|
| Security | L3 | 密钥面收紧 + 审计追责 |
| Auditability | L3 | 每次访问落库；含 auth_method / context / view |
| Compatibility | L2 | 破坏性：旧客户端无 header 且无 summary 将 403；前端同步改 |
| Performance | L2 | 审计 insert 异步失败不阻断主响应（try/except + log） |
| Privacy | L2 | UA/IP/Referer 作元数据；不记 Token 明文 |
