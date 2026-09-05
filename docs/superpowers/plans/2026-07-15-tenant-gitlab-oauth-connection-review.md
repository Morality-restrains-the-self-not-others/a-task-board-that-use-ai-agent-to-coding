# Review：租户级自建 GitLab OAuth 连接

**日期：** 2026-07-15  
**对照计划：** `2026-07-15-tenant-gitlab-oauth-connection-plan.md`

## 结论

**通过（可开 PR）** — 无阻断项；已做一处加固。

## 检查项

| 项 | 结果 |
|----|------|
| Go 单测 `taskGitOauth ./...` | ✅ 通过 |
| Archi `--loadModel` v30.archimate | ✅ Loaded model |
| 管理员门禁 PUT/DELETE | ✅ ensureTenantAdmin |
| GET 不回传 secret | ✅ 单测覆盖 |
| Intent→Event Upserted/Deleted | ✅ publish + conf/domain-events |
| 零新增 Python 公网 path | ✅ 仅扩展 providers |
| 日志禁 secret | ✅ 未打印 client_secret |

## 已修复

- PUT 更新时 `client_secret` 可空，保留原 Fernet 密文（避免每次改备注都强制重填密钥）

## 非阻断 / 后续

- Playwright 冒烟未跑（需完整 runAll + 自建 GitLab）；计划 Inc3
- 公网 SPA collectstatic：合入/部署前执行 `runall-lifecycle.sh build`
- 架构 v30 仍为 target；合入 main 后由 ship 切 current
