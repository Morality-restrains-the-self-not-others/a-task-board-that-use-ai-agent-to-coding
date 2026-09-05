# [运行时] work-panel 人过滤选项显示 user_id 前缀而非成员名

## 现象

`/tenant/{tenant}/work-panel` 打开「人」过滤下拉，`[data-alias="access-filter-person-option"]` 可见文案为类似 `85025667`（user_id 前 8 位），而非公司成员名称。

## 根因

1. `WorkspaceAccessSerializer.get_user_info` 未输出 `member_name`；`username` 仅读 `UserProfile.username`，缺失时 `_resolve_username` 回落 `user_id[:8]`。
2. 前端 `parseAccessFilterSubjects` 优先取 `username`，未优先 `member_name` / 协作人池。

## 修复

- 后端：用 `DisplayName(member_name → profile username → user_id[:8])` 填充 `username`，并显式返回 `member_name`。
- 前端：`resolveAccessPersonLabel` 优先 `member_name`，再协作人池，再 `username`。

## 验收

- Django：`tests/test_workspace_access_user_info_display_name.py`
- Vitest：`workPanelAccessFilter.test.js`（member_name 优先）
- Playwright：`WorkPanel.accessFilter.playwright.test.js`（mock username 为前缀仍显示 Alice/Bob）
