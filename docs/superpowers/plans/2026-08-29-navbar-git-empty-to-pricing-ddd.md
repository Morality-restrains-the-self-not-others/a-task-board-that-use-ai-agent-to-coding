# Navbar 无仓库跳价格页 — DDD

- **日期**: 2026-08-29
- **NFR**: `docs/superpowers/plans/2026-08-29-navbar-git-empty-to-pricing-nfr-clarification.md`

## 限界上下文

计费/GitLab 资源（taskBill，只读 GET）+ 前端导航（taskFE）。本增量只改前端适配器。

## 概念（无新实体）

- **Jumpable GitLab region**：`gitlab_web_url` 非空的已购/获赠区域。
- **Empty nav CTA**：jumpable 为空且列表 `ready` → 价格页。

## 事件

无。纯前端导航例外（见意图文档）。

## 端口

不新增。继续 `GET /api/tenant/{tid}/billing/gitlab-resources/`。
