# Application Integration v28 — Multi-Account Isolation

```mermaid
flowchart LR
  subgraph client [Browser]
    Vue["🟡 Vue activate/resolveSwitchHref"]
    Creds["authToken + userId"]
    Sess["sessionid cleared on switch"]
  end
  GW["task-gateway Token>Cookie"]
  DJ["🟡 Django Token>Session + me path check"]
  Auth[taskAuth activate-session]
  C["🟢 Constraint Token precedence"]

  Vue -->|RW| Creds
  Vue -->|clear| Sess
  Vue --> GW
  GW --> DJ
  GW --> Auth
  C -.-> DJ
  C -.-> GW
```

## Plateau / Gap

- **Plateau v24/v27**: 多账号切换已交付，但 Session 可压过 Token；租户 URL 残留
- **Gap**: A→B 切换后 B 仍可见 A 私有资源
- **Plateau v28**: Token 优先 + 清 sessionid + 离开租户 + me() 校验
