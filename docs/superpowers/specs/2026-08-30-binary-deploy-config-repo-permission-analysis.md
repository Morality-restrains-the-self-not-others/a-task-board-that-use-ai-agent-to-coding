# 角色权限分析：二进制部署与独立配置仓

- **Date:** 2026-08-30
- **Design:** `docs/superpowers/specs/2026-08-30-binary-deploy-config-repo-design.md`
- **Verdict:** 绿灯 ✅ — 无新租户 API；交付面是运维 ACL + 产物凭证，不是租户 RBAC

## 结论

无新 `page`/`region`、无新 HTTP 业务端点。权限边界在 **GitHub 仓 ACL、Packages 读权限、部署主机本地密钥文件**。租户用户不可见本增量。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| clone/push `daydaymoney-deploy` | 平台运维 / 发布角色 | System（工程 GitHub） | 写运行时 conf | GitHub org 私有仓 ACL | ✅ 充分 | 开发者默认无 write；与源码仓权限分离 |
| 读 GitHub Packages 产物 | CI 写、部署机读 | System | 读 ELF/dist | GitHub token / deploy token | ⚠️ 实施时 | 部署机用 **read-only** PAT/App；禁止把写 token 放配置仓 |
| `CONF_ROOT` / `DEPLOY_ROOT` | 主机进程环境 | System | 定位配置 | 仅本机 env | ✅ | 不可经公网 API 设置 |
| `config.local.yaml` | 主机运维 | System | 密钥 | gitignore + ADR-0046 | ✅ | 不进配置仓 Git |
| runAll 部署模式 start（无 build） | 本机 9999 session | System | 启停进程 | 既有 session/ownership | ✅ | 不扩大 9999 暴露面 |
| `releases.yaml` 改钉 | 配置仓写者 | System | 选产物版本 | PR + 私有仓 | ✅ | 单服务热修须 PR 说明兼容 |

## 角色建模

不新增租户角色。工程侧约定：

```
role: deploy_ops
display_name: 平台交付运维
permissions: [daydaymoney-deploy:write, github-packages:read, host-config.local:write]
scope: system
```

层级：`superuser`（源码+配置）⊃ `deploy_ops`（仅配置仓+拉产物）⊃ 普通开发者（仅源码仓）。

## 安全审计

- [x] 无新 IDOR URL / 无租户 path
- [x] 无跨租户数据
- [x] 生产密钥不进配置仓（ADR-0046 + ADR-0052）
- [x] 配置仓不托管租户 GitLab（gitService）
- [x] 产物不进 Git 历史
- [x] 部署机 read-only 拉包，避免写凭证落盘进仓

## 权限测试

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 无 CONF_ROOT 的开发树仍 FindMonorepoRoot | 开发者本机 | 启动 | 与现网一致 |
| CONF_ROOT 指向无密钥的 fixture | 测试 | FindConfigRoot | 成功且不读源码仓 conf |
| releases.yaml 钉与磁盘 sha 不一致 | 部署 sync | start | 不 exec 错版本；不 go build |

不新增租户 RBAC 测例。

## 风险评级

| 级别 | 项 | 缓解 |
|------|----|------|
| 中 | Packages token 权限过大 | 部署机只读；CI 用独立 write token |
| 低 | 配置仓误公开 | 创建时 `--private`；org 规则禁止 public |
