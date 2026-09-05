# 意图：内部换票路径按 Git site 寻址

- **日期**: 2026-08-22
- **状态**: 已设计（v97 target），待实现
- **设计**: `docs/superpowers/specs/2026-08-22-multi-comment-git-oauth-access-token-sharing.md`

## 背景与目标

`POST /api/internal/{github|gitlab}/oauth/access-for-user/` 用产品族当路径键，
多 GitLab 实例挤在同一条路径上。改为与审计列、浏览器 v2 回调一致的
**site = host[:port]**。

## 范围与边界

- 范围内：新路径 `POST /api/internal/gitsite/{site}/oauth/access-for-user/`；
  `{site}` URL 编码；YAML website → provider_key；三家调用方改默认路径。
- 范围内：旧 github|gitlab 与 git-oauth 别名路径保留（禁止删除，404 链）。
- 范围外：不按评论签发 GitLab OAuth/STS；不新增 Python API；无新领域事件。

## 验收标准

1. 新调用默认走 `/api/internal/gitsite/{site}/oauth/access-for-user/`。
2. `localhost:8012` 编码为 `localhost%3A8012` 仍命中 handler。
3. 旧路径仍 200 换票（别名）。
4. 响应仍禁止把 access_token 返回浏览器；internal 明文仅服务间。

## 业务意图 → 事件对照

| 业务意图 | 事件 | 例外理由 |
|---------|------|---------|
| 内部换票改路径 | 无 | 纯内部契约，无跨聚合副作用 |
