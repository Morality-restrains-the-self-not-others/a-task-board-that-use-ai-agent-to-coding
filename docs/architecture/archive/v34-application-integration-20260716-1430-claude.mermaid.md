# Application Integration v34 — Internal Repo Disk Size

```mermaid
flowchart LR
  Vue["Vue ProjectDetail\nGitReposSection"]
  GW[task-gateway\nJWT + tenant]
  Project["taskProjectService\nenrich disk_size_bytes\n+ 60s cache"]
  GitOAuth[gitoauth\nfetchGitAccessToken]
  GitLab["GitLab\n?statistics=true"]
  PR["project_repos\n(只读引用)"]
  C["Constraint\n仅 gitlab-local"]

  Vue -->|GET project| GW
  GW --> Project
  Project -->|R| PR
  Project -->|当前用户 token| GitOAuth
  Project -->|仅内部仓| GitLab
  C -.->|外部零调用| Project
```

## Plateau / Gap

- **Plateau v33 📦 archived**: 终态优雅通知容器收尾与回调释放
- **Gap closed**: 项目详情无法展示平台内部仓磁盘占用；外部仓不应展示
- **Plateau v34 ✅ current**: 详情 enrich `is_internal` + `disk_size_bytes`；60s 短缓存；前端条件徽章；无新公网 path
