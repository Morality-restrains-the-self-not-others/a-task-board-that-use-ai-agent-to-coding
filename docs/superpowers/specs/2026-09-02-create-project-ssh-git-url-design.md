# 创建项目 Git URL 支持 ssh:// 协议

- **Date:** 2026-09-02
- **Status:** accepted (goal-mode auto-adopt)
- **Architecture change:** none (no new services, APIs, or events)

## Problem

创建项目页 Git 仓库输入当前只接受：

- `https://host/path.git` / `http://…`
- SCP 风格 `git@host:path.git`

官方 Git SSH URI `ssh://git@host[:port]/path.git`（GitLab 非 22 端口常用）被前端 `isValidGitRepoUrl` 拒绝，展示「请输入有效的 Git 仓库 URL」。后端 `normalizeGitRepoURLForBranchLookup` 只把 `git@` 转成 HTTP(S)，`ssh://` 原样进入 OAuth/分支探测会失败。

用户在 `https://www.daydaymoney.com/tenant/…/create-project/` 看到该格式错误；期望支持 SSH 协议。

## Decision

**We will** treat `ssh://` as a first-class Git remote scheme:

1. Frontend format check accepts `ssh://` with hostname + non-empty path (keep `git@` and `http(s):`).
2. Backend normalizes `ssh://git@host[:port]/path` → HTTP(S) for OAuth token / branch / validate (same host mapping as `git@`).
3. Persist the original URL; clone via OAuth HTTPS or existing PEM SSH path (`git_clone_remote_for_ssh_pem` already preserves `ssh://`).
4. Update placeholders and format examples to include `ssh://`.

No new HTTP API. No Kafka event (URL is a field on existing project write).

## Alternatives Considered

| Option | Reject reason |
|-------|----------------|
| Only tell users to paste `git@` | Custom SSH ports cannot be expressed in SCP form; user asked for the protocol |
| Rewrite input to `git@` on blur | Drops `:port`; worse UX |
| New SSH-key-only clone pipeline | Product already clones via OAuth HTTPS; PEM path already exists |

## Consequences

- Custom-port GitLab SSH URLs become usable for create/validate/OAuth.
- Stored URL may be `ssh://` while runtime Git API uses HTTPS; match keys already equate them via host+path (`gitCloneRefMatchKey`).
- `git://` remains rejected (unauthenticated).

## 🕸️ Code Review Graph 分析

CRG `update --brief` ran at pipeline start (risk 0). No `.codegraph/` index; used `rg` on `isValidGitRepoUrl`, `normalizeGitRepoURLForBranchLookup`, `extractHostNetloc`, `looksLikeGitRepoRef`.

## 架构

无新组件。不新增 ArchiMate / ADR。
