# [运行时] GitHub `*.git` URL 被误判为 GitLab → 子仓报「GitLab repository not found」

- **日期**：2026-07-20
- **页面**：项目详情 → 子 Git 仓库 / 默认自动运行
- **项目示例**：`proj_-5458153230485381751`（`https://github.com/ruandao/somanyad.git`）

## 现象

1. Git 仓库行显示「已授权」+ `https://github.com/ruandao/somanyad.git`
2. 子 Git 仓库红字：`无法访问父仓库…（GitLab repository not found）`
3. 默认自动运行：`无法获取子 Git 仓库列表，自动运行无法启动…`
4. 公网 `GET https://api.github.com/repos/ruandao/somanyad` → **200**（仓本身可见）

## 根因

`ProviderResolver.IsGitLabRepo` 曾用「URL 以 `.git` 结尾」作为 GitLab 启发式，导致：

| URL | `IsGitHubRepo` | `IsGitLabRepo`（旧） | `fetchRepoRawFile` 路径 |
|-----|----------------|----------------------|-------------------------|
| `https://github.com/ruandao/somanyad.git` | true | **true（误）** | GitLab API → `GitLab repository not found` |
| `https://github.com/ruandao/somanyad` | true | false | GitHub API → OK |

`listNestedGitRepos` 只把 `isGitLab` 传给 raw-file 拉取，GitHub host 也被走 GitLab 探测，括号细节里出现「GitLab repository not found」。

## 快速复现 / 对比

```bash
# 旧逻辑下：带 .git 失败；不带 .git 成功（error 空）
curl -sS "http://127.0.0.1:8016/api/internal/nested-git-repos/?user_id=<uid>&repo_url=https://github.com/ruandao/somanyad.git"
curl -sS "http://127.0.0.1:8016/api/internal/nested-git-repos/?user_id=<uid>&repo_url=https://github.com/ruandao/somanyad"
```

## 修复（已合入 taskProjectService）

1. `IsGitLabRepo`：`github:` provider / `github.com` host 一律返回 false；`.git` 启发式仅留给未知 host。
2. `listNestedGitRepos`：`isGitHub` 时强制 `isGitLab=false`（防御）。

## 验收

```bash
curl -sS "http://127.0.0.1:8016/api/internal/nested-git-repos/?user_id=<uid>&repo_url=https://github.com/ruandao/somanyad.git" \
  | python3 -c 'import sys,json;d=json.load(sys.stdin); assert d.get("error")=="" or "GitLab repository not found" not in (d.get("error") or "")'
cd taskProjectService && go test ./src/ -count=1 \
  -run 'IsGitLabRepoDoesNotMisclassify|ProviderHeuristicMatrix|ListNestedGitReposGitHubDotGit'
```

矩阵覆盖（OPT-20260720-030）：`git@github.com:…`、自建 `https://git.example.com/…/*.git`、`listNestedGitRepos` 对 `*.git` 优先走 GitHub API。

页面硬刷新后：子仓区不再出现「GitLab repository not found」；无 `.gitmodules` 时为空列表而非红错。

## 前端可观测（附）

该红字曾缺失 `data-traceId`：`useProjectNestedGitRepos` 只写了 `nestedError` 文案、未从响应头提取 traceId，模板也未绑定。已补 `nestedErrorTraceId` →
`[data-testid="project-detail-nested-git-repos-error"]`，自动运行门禁提示同源透传。

## 与相关条目区分

- [43_nested_git_repos_empty_on_inaccessible_parent.md](./43_nested_git_repos_empty_on_inaccessible_parent.md)：真·父仓无权限 / App Contents 为空 →「无法访问父仓库」。
- 本条：父仓是 **公开 GitHub**，却因 **provider 误判** 走了 GitLab API。
