# 角色权限 — 部署机 9999 源码编译重启

- **Date:** 2026-09-02
- **Design:** `docs/superpowers/specs/2026-09-02-deploy-9999-source-compile-restart-design.md`

9999 是本机 ops 控制面（clone-run `DEPLOY_MODE=1`），不是租户产品面。不引入 RBAC 角色、不走 PDP、不跨租户。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `POST /api/precise-restart` | 能打开本机 9999 的运维/开发者 | System（部署根 + `SOURCE_ROOT`） | 编译 + 安装 + 重启已登记服务 | 既有 9999 本地 UI；bulk 互斥 | ✅ 充分 | 不新开端点 |
| `POST /api/build-all` | 同上 | System | 源码 `--all` 编译 + 安装；不杀进程 | 同上 | ✅ 充分 | 不新开端点 |
| 读 `$SOURCE_ROOT/.runall/precise_restart_services.txt` | 同上 | System | 读登记 | 文件本机路径 | ✅ | Agent 仍写源码仓 |
| `rsync conf-local/` | 同上 | System / 机密树 | 覆盖部署根 `conf-local/` | 同机假设 | ✅ | 禁止把内容打进日志 |
| `install-local-artifacts` | 同上 | System / 产物 | 写 `bin/` | copy-then-mv | ✅ | 失败保留 last-good |

## 建模结论

无新角色。前端按钮既有 anti-replay / bulk queue（元规则 52）。`SyncMonorepoConf` 在 `DEPLOY_MODE` 下禁止把源码 `conf/` 叠到部署 `conf/`。
