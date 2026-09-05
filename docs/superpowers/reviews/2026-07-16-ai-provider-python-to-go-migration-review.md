# Review：ai-provider Python→Go

**日期：** 2026-07-16  
**结论：** 通过（可开 PR）；已知缺口见下

## 对照计划

| 项 | 状态 |
|----|------|
| taskAiProvider 脚手架 | ✅ |
| Auth JWT/SSO/OIDC | ✅（OIDC id_token 未做完整 JWKS 校验，与内网信任模型一致，后续可加固） |
| Public APIs | ✅ 主路径 |
| Vendor/Admin CRUD | ✅ 主路径；部分 DRF 细粒度字段/分页未 1:1 |
| Cloud proxy | ✅ |
| SPA | ✅ |
| runAll 切流 | ✅ |
| Python 清理 | ✅ 仅留 README |
| 测试 | ✅ go test；health 冒烟 |

## Log / Intent→Event 审计

- 入站：tracelog Middleware
- 事件：SSO/OIDC/审批结构化 logInfo event=…

## 非阻塞优化

- OIDC JWKS 验签、registry manifest resolve、userdata 完整 wrapper 脚本、DRF 分页格式
- Playwright 文案仍含历史「django8010」文件名（行为已指向 :8010 Go）

## Follow-up (ship)

- 补齐 `taskAiProvider/scripts/runall-stop.sh`（根 `.gitignore` 曾忽略 `scripts/`，已加例外）
- 架构 v32 `@status: current`；v31 archived
- PRs: ram-work#2 docs#17 conf#6 db#5 task2app#23
