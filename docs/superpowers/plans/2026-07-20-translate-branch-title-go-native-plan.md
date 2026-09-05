# Plan: translate-branch-title Go native

## Tasks

- [x] `fanyi_agent.go`：加载 conf + DirectClient 调用
- [x] `translate_branch_title.go`：handler 去掉 djangoPost
- [x] 单测：中文成功 / 配置缺失 / 上游错误 / sanitize
- [x] 削 `utility_handlers.go`（gitlab 段拆出）
- [x] OpenAPI / api-route-to-owner / 架构 v42
- [x] 重启 taskProjectService 手工验收中文标题（200 used_ai=true）
- [x] PR

## 事件契约

无（工具 API 例外）。
