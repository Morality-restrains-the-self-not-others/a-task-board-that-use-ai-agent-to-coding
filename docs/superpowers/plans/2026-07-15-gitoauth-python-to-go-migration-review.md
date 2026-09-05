# Review：gitOauth → taskGitOauth

**日期：** 2026-07-15  
**审查人：** claude (0-auto-flow Step 9)

## 结论

**通过（修复 Critical 后）** — 可进入 Ship。

## Critical（已修复）

| ID | 问题 | 修复 |
|----|------|------|
| C1 | GitHub callback 存 `provider=github`，消费者查 `github:github-official` 失败 | 写入改用配置 `ProviderKey`；`ExpandProviderKeys` 双向别名 |
| C2 | 契约测不足 | 新增 alias access/summary 单测 + ExpandProviderKeys 单测 |

## Important（已处理/接受）

| 项 | 状态 |
|----|------|
| access 响应补 `github_user_id` / `expires_in` | 已修 |
| browser Method=GET 校验 | 已修 |
| companion `runAll.yaml.ai.md` / value-stream | 已改 taskGitOauth |
| OpenAPI 仅 stub summary | 接受（路径清单齐全；schema 后续迭代） |
| Session 与 Django 不兼容 | 设计已接受 |

## Log / Intent 审计

- 换票失败、decrypt 失败有 `logWarn`（无 secret）
- 意图事件：首期结构化日志/审计表，MQ 书面例外（与 Python 一致）— 见 `docs/intents/backend/gitoauth_go_migration.intent.md`

## 验证

```
cd taskGitOauth && go test ./...   # PASS
curl :18002/api/health/            # ok
runAll language/build tests        # PASS
```
