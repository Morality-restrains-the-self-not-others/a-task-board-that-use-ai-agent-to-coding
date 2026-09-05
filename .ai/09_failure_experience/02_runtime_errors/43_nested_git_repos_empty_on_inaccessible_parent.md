# [运行时] 父仓不可访问时子 Git 仓库显示「未发现子仓库」

- **日期**：2026-07-17
- **页面**：项目详情 → 子 Git 仓库（`data-testid="project-detail-nested-git-repos-empty"`）
- **项目示例**：`proj_-5247879312070945751`（`https://github.com/task2money/ram-work`）
- **重置敏感**：是 — **DB / taskGitOauth 重置、用户重新 OAuth、换 GitHub App 权限** 后极易复现

## 现象

父仓根目录已有 `.gitmodules`（含数十个子模块），页面却显示灰色文案「未发现子仓库」，或（修复后）红色「无法访问父仓库…」。绑定徽章仍可能显示「已授权」。

### 2026-08-12 复验（daydaymoney App）

- **页面**：`proj_-2823506342987866751` / `https://github.com/task2money/ram-work`
- **traceId**：`a6ef30d9-1d20-4301-a3dd-c6aa9d2a044d`
- **Loki 时间线**：task-auth forward-auth 200 → task-git-oauth `access-for-user` 200（1ms，有票）→ task-project-service `nested-git-repos` 200（~5.4s，业务 error）
- **当前 App**：`daydaymoney`（`permissions.contents=write`）——**不是**权限为空；根因是 **未安装到组织**
- **实证**：用户 `ghu_` 调 `GET /user` → 200（`ruandao`）；`GET /repos/task2money/ram-work` → 404；`GET /user/installations` → `total_count=0`；`GET /orgs/task2money/installations` → 空。个人 `gh` 对同仓 admin 可见。

## 根因

1. `listNestedGitRepos` 用用户 OAuth token 拉远端 `.gitmodules`；GitHub 对**无权限的私有仓**统一返回 **404**（与「文件不存在」相同）。
2. **旧行为**：`fetchRepoRawFile` 将 main/master 上的 404 **静默当成空文件** → `nested_repos: []` 且 `error: ""` → 前端走 empty 态。
3. **徽章「已授权」≠ 能读 Contents**：`token_available` 只表示 OAuth 绑定/换票成功；子仓发现另做 repo/Contents probe。
4. GitHub App 侧两类失败都会导致父仓 404：
   - 注册 **Permissions 为空**（历史 `daydaymoney`：`GET /apps/daydaymoney` → `permissions: {}`）
   - **未安装到组织/账号**或安装范围不含该仓（当前 `daydaymoney`：权限已有，但 `task2money` 安装数为 0）
5. YAML 里的 `scope: repo read:user` **不能**代替 App 控制台 Permissions + 安装范围（`ghu_` 以二者为准）。

## 服务重置后为何会再碰到

| 重置动作 | 会丢什么 | 表现 |
|---------|---------|------|
| `db/git-oauth` 或 credential 表清空 | 用户绑定 / 可用 access token | 未授权，或重绑后仍无 Contents |
| 用户在页面「重新 OAuth」 | 可能用空权限 App 换到新的 `ghu_`，覆盖此前可用的 opaque token | 父仓再次 404 |
| 仅重启进程、未改 App 权限 | 内存 cache 清空；若库内仍是无权限 `ghu_` | 同上 |
| 开发库整体 reset（见 `docs/superpowers/specs/2026-05-31-dev-database-reset-design.md`） | 绑定全无 | 需重新绑定 + 确认 App 权限 |

**结论**：代码侧「勿把仓不可见当成 empty」在重置后仍有效；**能列出子仓**仍依赖 GitHub App 具备 Contents: Read 并安装到组织。临时注入 CLI `gho_` **不会**随库/重绑保留。

## 快速诊断（重置后优先跑）

```bash
# 1) 当前 App 权限 + 组织是否已安装
env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u all_proxy \
  gh api apps/daydaymoney --jq '{slug,permissions}'
gh api orgs/task2money/installations --jq '{total_count, apps: [.installations[].app_slug]}'

# 2) 当前用户票能否看见父仓（用 access-for-user 拿到的 ghu_）
#    /user → 200 且 /repos/... → 404 且 /user/installations.total_count=0 → 未安装 App
curl -sS -o /dev/null -w 'repos:%{http_code}\n' \
  -H "Authorization: Bearer <access_token>" -H "Accept: application/vnd.github+json" \
  https://api.github.com/repos/task2money/ram-work
curl -sS -H "Authorization: Bearer <access_token>" -H "Accept: application/vnd.github+json" \
  https://api.github.com/user/installations | python3 -c 'import sys,json;print(json.load(sys.stdin).get("total_count"))'

# 3) 发现 API（应：有列表，或 error 含「无法访问父仓库」，禁止无 error 的空列表）
curl -sS "http://127.0.0.1:8016/api/internal/nested-git-repos/?user_id=<uid>&repo_url=https://github.com/task2money/ram-work" \
  | python3 -c 'import sys,json;d=json.load(sys.stdin); print("error=",d.get("error")); print("count=",len(d.get("nested_repos") or []))'
```

日志线索：`logs/task-project-service.log` 中  
`nested-git-repos project=proj_… parent=https://github.com/task2money/ram-work`。

## 修复（代码，已合入意图）

1. **taskProjectService**：全部 ref 404 后 probe 父仓；父仓不可见 →「无法访问父仓库…」；优先 `default_branch`。
2. **taskGitOauth**：`gho_` / `ghp_` / `github_pat_` 与 `ghu_` 一样按直通 access token 使用。
3. **前端**：子仓错误节点须挂 `data-traceId`（`useProjectNestedGitRepos.nestedErrorTraceId` →
   `project-detail-nested-git-repos-error`；自动运行门禁提示透传同值）。此前曾漏接，见 2026-07-20 修复。

## 恢复清单（重置后要看到子仓列表时）

1. 确认当前 App 为 **daydaymoney**：`gh api apps/daydaymoney --jq '{slug,permissions}'`（应含 `contents`）。
2. 安装 / 调整安装范围：https://github.com/apps/daydaymoney/installations/new  
   - 勾选组织 **task2money**（或至少包含 `ram-work` 的仓库集）。  
   - 验收：`gh api orgs/task2money/installations --jq '.total_count'` ≥ 1。
3. 在 SaaS 个人资料 / 项目页对 GitHub **重新 OAuth 授权**（使 installation 进入用户票；`GET /user/installations` 应非 0）。
4. 硬刷新项目详情 → 应出现 `data-testid="project-detail-nested-git-repo-row"`（ram-work 预期约 30+ 行）。
5. 若仍失败：用上面「快速诊断」区分「未绑定」vs「未安装/票无 Contents」vs「父仓真无 `.gitmodules`」。

Companion：`conf/auth/git-oauth/providers/github.ai.md`  
相关 OPT：`OPT-20260812-025`（安装 App）；历史 `OPT-20260717-048`（daydaymoney Contents，已 cancelled）

## 验收

```bash
curl -sS "http://127.0.0.1:8016/api/internal/nested-git-repos/?user_id=<uid>&repo_url=https://github.com/task2money/ram-work" \
  | python3 -c 'import sys,json;d=json.load(sys.stdin); assert len(d["nested_repos"])>0 or "无法访问父仓库" in (d.get("error") or "")'
```

页面硬刷新后应出现 `project-detail-nested-git-repo-row`，不再是误报的 `…-empty`。
