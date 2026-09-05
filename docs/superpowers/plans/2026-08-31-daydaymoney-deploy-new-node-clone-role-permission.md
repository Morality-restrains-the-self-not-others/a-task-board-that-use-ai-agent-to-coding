# Role-Permission: 新节点 clone-run（只拷 conf-local）

- **日期**: 2026-08-31
- **设计**: `docs/superpowers/specs/2026-08-31-daydaymoney-deploy-new-node-clone-design.md`

## 结论

无新增 HTTP/API。主体是**主机运维员**持有 `conf-local/` 与 GitHub token。不引入租户角色。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 手工放置 `conf-local/` | 运维员 | System | write | gitignore + 不进 Release | ✅ | 禁止提交 PEM |
| `up.sh` overlay | 运维员 | System | write | 仅 rsync conf-local | ✅ | 删除 taskGateway/db 树 overlay |
| `taskGateway setup_tls` | 进程 | System | write certs | DEPLOY_MODE 缺 PEM 失败 | ✅ 本迭代补 | 禁止 openssl 写死旧 IP |
| `taskAuth` OIDC 签名钥 | 进程 | System | read PEM | DEPLOY_MODE 缺钥失败 | ✅ 本迭代补 | 禁止 mint 新钥破坏 SSO |
| `gh release download` | 运维员 + token | System | read artifacts | 私有仓需 GITHUB_TOKEN | ✅ | 失败须可见，勿空 bin 当成功 |

无新角色。IDOR/租户越权不适用。
