# Companion：daydaymoney GitHub App

当前 SSOT 凭据：`http-github-com--app-daydaymoney.yaml`（`service_provider: github-official-daydaymoney`，client_id `Iv23li4xi6ZBcq1LKZk6`）。
旧 App `daydaymoney` 已退役；勿再按空 Permissions 的 daydaymoney 排障。

## 「已授权」却「无法访问父仓库」

两套独立判定，**可以同时为真**：

| UI | 判定 | 含义 |
|----|------|------|
| 徽章「已授权」 | `token_status=token_available` | 用户已完成 OAuth，库内有可用 `ghu_` 票（`access-for-user` 成功） |
| 红字「无法访问父仓库」 | Contents / repo API 对父仓 **404** | 该票看不到私有仓（GitHub 对无权限私有仓统一 404） |

常见根因（2026-08-12 已用 trace `a6ef30d9-…` 复验）：

1. **App 未安装到父仓所属组织**（`GET /user/installations` → `total_count=0`；`GET /orgs/<org>/installations` → 空）——即使 App 注册了 `contents: write`
2. 安装范围未包含该仓库
3. 权限变更后用户未重新 OAuth（旧票未带上新 installation）

个人 `gh` / PAT 能看仓 **不能** 说明 `ghu_` 票也能看仓。

**恢复**：

1. 组织管理员打开 https://github.com/apps/daydaymoney/installations/new （勾选组织 **task2money**，至少包含 `ram-work`）
2. 在 SaaS 项目页对 GitHub **重新 OAuth**
3. 硬刷新项目详情 → 子仓列表应出现

**SSOT 排查**：`.ai/09_failure_experience/02_runtime_errors/43_nested_git_repos_empty_on_inaccessible_parent.md`

YAML 里的 `scope: repo read:user` **不能**代替 GitHub App 控制台 Permissions；`ghu_` 以 App 注册权限 + **安装范围**为准。
