# 测试意图：租户自建 GitLab OIDC SSO（后端）

- **日期**: 2026-08-25
- **对应功能意图**: `docs/intents/backend/tenant_gitlab_oidc_sso.intent.md`

## 用例

1. 成员 + tenant client → 302 带 `code=`
2. 非成员 / 超管非成员 / 成员服务 5xx → 302 `error=access_denied`
3. `gitlab-git-service*` 不走租户成员闸门（仍走区域闸门）
4. PUT 首次返回 `client_secret`；第二次 PUT 不返回 secret
5. GET 响应 JSON 无 `client_secret` 键或值为空
6. rotate 同 Idempotency-Key 重放不二次换密展示
7. DELETE 后再 GET `configured=false`
8. base_url 与 Path A 不一致 → 400
9. 事件：签发/轮换/吊销各至少一次 publish（可用 fake writer）
10. OmniAuth 片段含 `/api/oidc/{tid}/authorize` 且 `discovery: false`；错租户 path authorize → 400 `unauthorized_client`
11. 公网 HTTP base_url PUT → 200 且返回一次性 secret；非 http/https scheme 仍 400

## 文件

- `taskAuth/domain/tenant_gitlab_oidc_sso_test.go`
- `taskAuth/src/oidc_tenant_sso_gate_test.go`
- `taskAuth/src/handlers_tenant_gitlab_oidc_sso_test.go`
