# 测试意图：内部换票路径按 Git site 寻址

对应：`docs/intents/backend/gitoauth_access_for_user_gitsite_path.intent.md`

| 编号 | 场景 | 前置 | 操作 | 期望 |
|------|------|------|------|------|
| T1 | 新路径命中 | mux 注册 | POST `/api/internal/gitsite/github.com/oauth/access-for-user/` 空 user_id | 400 bad user_id，非 404 |
| T2 | 端口编码 | site=`localhost:8012` | POST `.../gitsite/localhost%3A8012/oauth/access-for-user/` | 命中同一 handler |
| T3 | 旧路径别名 | 既有 gitlab 路径 | POST `/api/internal/gitlab/oauth/access-for-user/` | 非 404 |
| T4 | 调用方默认 | project/credential/cloud 客户端 | 构造 URL | 含 `/gitsite/`，不含默认 `/github/` 或 `/gitlab/` 族路径 |

- 实现测：`taskGitOauth/src/internal_path_contract_test.go`（扩展）
- 调用方测：各 `gitoauth_client` URL 断言
