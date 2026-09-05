# Value Stream: relay bootstrap GitLab 克隆 HTTP 认证修复

> 设计：`docs/superpowers/specs/2026-05-27-relay-bootstrap-gitlab-clone-auth-design.md`

## Value Summary

relay 直启后 onlineServiceJS bootstrap 能使用正确的 GitLab HTTP Basic 用户名（oauth2）完成克隆，不再因 namespace 路径段误作用户名而 exit 128。

## End-to-End Flow

[直启启动] → [token-exchange] → [task-detail + repo-clone-credentials] → [bootstrap git clone with oauth2] → [任务引导完成]

## Value Increments

### Increment 1: 凭证 payload 扩展 + bootstrap 用户名对齐（薄切片）

- 后端 `repo-clone-credentials` 返回 `provider` + `git_http_username`
- `buildHttpAuthFromRepoCredential` 优先使用凭证字段
- Django + Node 单测

### Increment 2: E2E 回归 + outbound 日志

- 更新 bootstrap e2e 断言 oauth2
- bootstrap 日志记录 provider/username（不含 token）

## Test Mapping

- `tests/test_container_runtime_tokens.py`
- `trae-agent/onlineServiceJS/src/bootstrap.cloneCredentials.test.mjs`
- `trae-agent/onlineServiceJS/e2e/bootstrap-clone-host-alias-credential.api.spec.mjs`
