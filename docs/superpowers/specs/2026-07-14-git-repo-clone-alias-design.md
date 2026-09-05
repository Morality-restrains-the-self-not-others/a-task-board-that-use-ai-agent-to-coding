# Git 仓库克隆别名（clone_alias）

日期：2026-07-14  
状态：已批准（goal-mode 自动采用）

## 问题

创建项目页仅收集仓库 URL；容器克隆时目录名一律由 `repoDirNameFromUrl(url)` 推导。多仓同名或用户希望固定目录名时无法指定。

## 目标

1. 创建/编辑项目时，每个 Git 仓库可填可选「别名」
2. 别名持久化到 `project_repos.clone_alias`
3. 任务容器 bootstrap / reclone 优先用别名作为克隆目录名；空则保持现有 URL 推导
4. trae-agent（onlineServiceJS）完整支持该字段

## 非目标

- 不改 OAuth / validate-git-repo 语义（仍按 URL）
- 不改 `repoMatchKey`（仍以 URL 为键）
- 不强制历史数据回填别名
- 不新增服务或 Django 新路由

## 方案（采用）

### 字段

| 层 | 名称 | 说明 |
|----|------|------|
| DB `project_repos` | `clone_alias TEXT DEFAULT ''` | 空 = 运行时推导 |
| API 请求 `git_repos[]` | `string` 或 `{url, clone_alias}` | 向后兼容 |
| API 响应 | 保留 `git_repos: string[]` + 新增 `git_repo_entries: [{url, clone_alias}]` | 旧客户端不破 |
| task-detail / snapshot | 同 `git_repo_entries` | 容器消费 |
| UI | 别名输入框（可选） | CreateProject + ProjectEdit |

### 校验

- 别名可选；非空时 sanitize：`[^A-Za-z0-9._-]` → `-`，去首尾 `.`/`-`/`_`；sanitize 后为空则视为未填
- 同项目内非空别名唯一（前端提示 + 后端拒绝或自动 `_2` 后缀由克隆侧兜底）
- 匹配/OAuth 仍用 URL

### 克隆

```
dirName = sanitize(clone_alias) || repoDirNameFromUrl(url)
// 再走现有 reservedNames / name_2 去重
```

### 架构影响

无拓扑变更：仅扩展既有 `project_repos` 列与契约字段。不更新 `docs/architecture/` 视图。

## 测试

- Go：create/update/load 含 `clone_alias`；string 形态仍可用
- JS：`collectRepoCloneJobs` / 目录名优先别名
- 前端：提交 payload 含对象项；空别名仍可提交 string[]
- Playwright（增量）：CreateProject 行可见别名输入

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-14 | goal-mode 初版并自动采用 |
