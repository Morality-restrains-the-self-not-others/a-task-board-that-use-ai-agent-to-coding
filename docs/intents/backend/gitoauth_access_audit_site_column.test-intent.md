# 测试意图：访问令牌审计表 site 存域名+端口

## 对应功能意图

`docs/intents/backend/gitoauth_access_audit_site_column.intent.md`

## 用例

| ID | 场景 | 步骤 | 期望 |
|----|------|------|------|
| T1 | 写入 site | InsertAccessAudit(`github.com`) | `SELECT site` 读回 `github.com` |
| T2 | website → host | `https://github.com` | `github.com`（无 :443） |
| T3 | website → host:port | `http://localhost:8012` | `localhost:8012` |
| T4 | provider_key 解析 | 配置 website 后 token-use-report | 审计行 site 为 host[:port] |
| T5 | 解析失败 | 无匹配 provider 配置 | token-use-report 非 200，不写入 provider_key |

## 可执行测试

- `taskGitOauth/infrastructure/access_audit_site_test.go`
- `taskGitOauth/src/access_audit_site_test.go`
- `taskGitOauth/src/internal_handlers_test.go`（`TestHandleTokenUseReportSuccess`）
