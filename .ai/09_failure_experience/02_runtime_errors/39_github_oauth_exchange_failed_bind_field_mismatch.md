# [运行时] GitHub OAuth 换票后 bind 字段名错误 → exchange_failed

## 基本信息

- 案例编号：FE-20260717-GITOAUTH-BIND-FIELD
- 录入日期：2026-07-17
- 最后更新：2026-07-17
- 关联服务：taskGitOauth（:8002）、saas-backend Django `GithubAppInternalBindView`

## 失败现象

- 页面：`https://www.daydaymoney.com/profile/git-site-oauth/?github=exchange_failed`
- 可见文案：「授权失败：无法与 GitHub 交换令牌」
- GitHub 授权页可完成；回调回到本站后失败
- 前置：`bad_state` 已用签名 `state` 修复后，用户进入本错误码

## 根因

1. `taskGitOauth` 回调在 **GitHub `authorization_code` 换票成功**、凭据已写入 SQLite 后，调用 Django：
   `POST /api/accounts/github/internal/bind/`
2. Go 侧 payload 使用库内字段名 `remote_user_id` / `remote_login`
3. Django `GithubAppInternalBindView` 只认 `github_user_id`（见 `github_app_views.py`）→ **HTTP 400** `bad github_user_id`
4. Go 将任意非 200 bind 映射为回调码 **`exchange_failed`**，前端展示误导性「无法与 GitHub 交换令牌」

复现对照：

```bash
# 错误字段 → 400
curl -sS -X POST 'http://127.0.0.1:8001/api/accounts/github/internal/bind/' \
  -H "X-GitOauth-Bridge-Secret: $SECRET" -H 'Content-Type: application/json' \
  -d '{"user_id":<uid>,"remote_user_id":"12345"}'
# → {"detail":"bad github_user_id"}

# 正确字段 → 200
curl -sS -X POST 'http://127.0.0.1:8001/api/accounts/github/internal/bind/' \
  -H "X-GitOauth-Bridge-Secret: $SECRET" -H 'Content-Type: application/json' \
  -d '{"user_id":<uid>,"github_user_id":"12345"}'
# → {"ok":true}
```

## 修复

1. Go bind payload 使用 `github_user_id` / `github_login`（并可选带 `access_token`）
2. 单测锁定字段契约：`taskGitOauth/src/github_bind_payload_test.go`
3. 重建并重启 `task-git-oauth`

## 相关续发（2026-07-17 晚）

同一页面文案再次出现时，trace `24f9ed900ccb03b33ceddf65` 对应日志为：

`Post https://github.com/login/oauth/access_token: context deadline exceeded`（约 30s），属**出站网络超时**，非 bind 字段。

另：`profile["id"]` 经 `fmt.Sprint(float64)` 会变成 `1.321779e+06`，Django 拒收 → `bind_http_400`。须用 `FormatRemoteUserID`。

加固：OAuth HTTP Client 60s + dial/header 超时 + 网络错误一次重试；`Proxy: nil`。

## 预防

- Python→Go 迁移时对照 Django 内部 API **请求体字段名**，勿仅对齐 DB 列名
- JSON `map[string]any` 数字 ID **禁止** `fmt.Sprint`，用十进制格式化
- bind / 换票失败应打不同回调码或日志（避免一律 `exchange_failed`）
- 变更后用上述 curl 契约探活 + `cd taskGitOauth && go test ./...`；出站 GitHub 探活勿经死 SOCKS
