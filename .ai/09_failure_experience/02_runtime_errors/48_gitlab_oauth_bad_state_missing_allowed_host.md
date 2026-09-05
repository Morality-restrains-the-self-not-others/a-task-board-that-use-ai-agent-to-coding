# [运行时] GitLab OAuth start → bad_state / missing_allowed_host（provider YAML 被删）

## 基本信息

- 案例编号：FE-20260718-GITOAUTH-MISSING-ALLOWED-HOST
- 录入日期：2026-07-18
- 最后更新：2026-07-18
- 关联服务：taskGitOauth（:8002）、前端项目详情页 Git 仓 OAuth

## 失败现象

- 页面：`/tenant/{id}/projects/{proj}/`（Git 仓列表 OAuth 按钮）
- 可见文案：`无法启动 GitLab 授权（bad_state） [debug: missing_allowed_host]`
- `GET /api/accounts/gitlab/oauth/start-from-gateway/` → **HTTP 503**
- 健康检查：`gitlab_provider_config_count: 0`
- 日志：`未找到 provider_key=gitlab:daydaymoney-gitlab 对应的配置`

## 根因

1. `conf/auth/git-oauth/providers/` 下三份 GitLab provider YAML 在提交
   `39b89e6`（「清理未用 oauth provider 文件」）中被误删：
   - `http-gitlab-daydaymoney-com.yaml`（`daydaymoney-gitlab`）
   - `http-localhost-8012.yaml`（`gitlab-local`）
   - `http-synology-gitlab.yaml`（`synology-gitlab`）
2. taskGitOauth 仅从该目录加载 `gitOauth` provider catalog；目录只剩 GitHub →
   `ResolveGitLabAuthorizeContext` 找不到 `Website` → 返回 `missing_allowed_host`
3. Django / task-credential 侧同名目录仍保留 YAML，故前端 catalog 仍能解析出
   `service_provider=daydaymoney-gitlab`，但 Go 侧无对应配置 → 启动授权失败

## 修复

1. 从 `39b89e6^` 恢复上述三份 YAML（`service.host: 0.0.0.0`）
2. 重启 `taskGitOauth`（`bash taskGitOauth/run.sh start`）
3. 验收：
   ```bash
   curl -sS http://127.0.0.1:8002/api/health/ | jq .gitlab_provider_config_count
   # → 3
   curl -sS -H 'X-User-Id: 1' -H 'Accept: application/json' \
     'http://127.0.0.1:8002/api/accounts/gitlab/oauth/start-from-gateway/?next=/x&return_key=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa&repo_url=https%3A%2F%2Fgitlab.daydaymoney.com%2Fgroup%2Frepo.git&service_provider=daydaymoney-gitlab'
   # → 200 + authorize_url（无 missing_allowed_host）
   ```

## 预防

- **禁止**将 `conf/auth/git-oauth/providers/*gitlab*` 当作「未用文件」删除；
  该目录是 taskGitOauth 的 provider SSOT（与 Django/task-credential 目录须对齐）。
- 清理前先看健康字段：`gitlab_provider_config_count` 应为预期 GitLab 条目数（当前 ≥1）。
- 伴读：`conf/auth/git-oauth/ai.md`
