# Application Integration v24 — Navbar Multi-Account Switcher

```mermaid
flowchart LR
  subgraph client [Browser]
    Vue["🟡 Vue Navbar/Login"]
    Slots["🟢 savedAccounts localStorage"]
  end
  GW[task-gateway]
  Auth["🟡 taskAuth activate-session"]
  DJ[saas-backend enrich-login]
  Tok[(accounts_customtoken)]

  Vue -->|RW| Slots
  Vue -->|POST activate-session| GW
  GW --> Auth
  Auth -->|validate| Tok
  Auth -->|internal enrich-login| DJ
```

## Plateau / Gap

- **Plateau v21**: 单会话 Token + 昵称链到 profile
- **Gap**: 无法在导航栏切换多个已登录账号
- **Plateau v24**: 多账号槽 + activate-session + Navbar 下拉
