# 014 — 测试意图：人过滤选项展示成员名

## 单元

| ID | 断言 |
|----|------|
| T1 | `WorkspaceAccessSerializer`：有 `member_name` 时 `user_info.username` / `member_name` 均为成员名 |
| T2 | 成员名为空时回落 `UserProfile.username` |
| T3 | 均无时回落 `user_id[:8]` |
| T4 | `parseAccessFilterSubjects` / `resolveAccessPersonLabel`：`member_name` > 协作人池 > `username` |

## Playwright

| ID | 断言 |
|----|------|
| E1 | mock permissions 中 `username` 为 id 前缀、`member_name` 为 Alice/Bob 时，`access-filter-person-option` 文案为 Alice/Bob |

测例路径：

- `task2app/Saas_project/tests/test_workspace_access_user_info_display_name.py`
- `task2app/front_project/app/src/utils/workPanelAccessFilter.test.js`
- `task2app/playwright/front_project/tests/WorkPanel.accessFilter.playwright.test.js`
