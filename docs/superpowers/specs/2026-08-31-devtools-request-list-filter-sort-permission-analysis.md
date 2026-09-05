# 角色权限分析：DevTools 请求列表过滤与排序

- **Date:** 2026-08-31
- **Design:** `docs/superpowers/specs/2026-08-31-devtools-request-list-filter-sort-design.md`

## 结论

**无新服务端端点、无新 Chrome 权限、无跨租户数据。** 过滤/排序只作用于当前 DevTools 会话已捕获的请求副本。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `queryRequestList` 本地过滤排序 | 本机已打开该面板的用户 | 本机浏览会话 Network 快照 | read | 仅 DevTools 面板可见 | ✅ 充分 | — |
| 类型芯片 / 表头排序 UI | 同上 | 同上 | UI | 无写 API | ✅ 充分 | Anti-Replay-OK：纯 UI |
| 既有 `createTask` | 已登录用户 | Workspace/Task | write | 既有登录 + 工作空间表单 | ✅ 充分 | 本增量不改 |

## 角色建模

不引入新角色。不新增 `manifest.json` permissions。

## IDOR / 数据泄露

请求列表来自 `chrome.devtools.network`（当前 inspected tab）。过滤不扩大可见面。禁止把 URL/body 写入结构化日志。

## 按钮防重放

排序/过滤按钮无 POST。注释 `Anti-Replay-OK: local view query only`。
