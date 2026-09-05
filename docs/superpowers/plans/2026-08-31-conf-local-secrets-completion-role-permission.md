# Role-Permission: conf-local 唯一 overlay

- **日期**: 2026-08-31
- **结论**: 无新 HTTP/RPC 端点；无角色/权限模型变更。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `ReadAppConfig` / `load_app_config` 读 `conf/` + `conf-local/` | 部署机进程 / 运维 | System（主机文件） | read | 文件系统权限 + gitignore | ✅ 充分 | 不把 `conf-local/` 暴露给租户 API |
| CI `check_conf_local_secrets.py` | CI / 开发者 | System | read 已跟踪树 | git ls-files | ✅ 充分 | 增 `*Pwd.md` 失败 |
| 抽取脚本写 `conf-local/` | 本机运维 | System | write | gitignore | ✅ 充分 | 禁止打印密钥值 |

无新角色。`python_api_approval: not_applicable`。
