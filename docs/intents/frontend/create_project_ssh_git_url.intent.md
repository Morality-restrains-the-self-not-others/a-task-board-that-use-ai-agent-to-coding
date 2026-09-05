# 功能意图：创建项目支持 ssh:// Git URL

## 用户故事

作为租户成员，我在创建（或编辑）项目时粘贴 Git 官方 SSH URI（`ssh://git@host[:port]/path.git`），系统应接受该协议，而不是只允许 `https://` 与 `git@host:path`。

## 验收标准

1. `ssh://git@github.com/owner/repo.git` 不出现「请输入有效的 Git 仓库 URL」。
2. 带自定义端口的 `ssh://git@host:2222/group/project.git` 格式合法。
3. 后端校验/OAuth/列分支将 `ssh://` 规范为 HTTP(S)，与现有 `git@` 行为一致。
4. 项目持久化原始 `ssh://` 字符串。
5. `https://` 与 `git@host:path` 行为不变；`ftp://`、无路径的 `ssh://host` 仍拒绝。

## 范围

- taskFE `gitRepoUrlUtils` / CreateProject / ProjectEdit 格式校验
- taskProjectService `normalizeGitRepoURLForBranchLookup`
- taskProjectService `normalizeGitRepoURLForBranchLookup`
- 克隆进度文案识别 `ssh://`（`looksLikeGitRepoRef`）

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 例外理由 |
|---------|--------|--------|----------|
| 接受 ssh:// 作为仓库 URL | — | — | 字段校验，无新业务状态 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 初版 |
