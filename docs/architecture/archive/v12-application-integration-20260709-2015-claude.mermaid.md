# v12 application-integration — relay clear-logs (path scope)

```mermaid
flowchart LR
  Vue["Vue Frontend"] -->|POST .../relay-to-trae/clear-logs/| APISIX["task-gateway APISIX"]
  APISIX --> CGW["taskContainerGateway"]
  CGW -->|POST /v1/tenant/t/workspace/w/task/task/clear-logs| RELAY["go-relay :8797"]
  RELAY -->|clears state.Logs / taskLogs / LogCursor| BUF["Startup log buffer"]
```

- Plateau v11 → Gap: server log buffer not cleared on UI clear
- WorkPackage: path-scoped clear-logs
- Plateau v12: clear-logs path contract
