# Ship Report: 公司切换后项目列表为空 — 修复

> Pipeline: 1→3→4→5→6→7→8→9 (DDD skipped)

## 已修改文件

| 文件 | 改动 |
|------|------|
| `Navbar.logic.vue` | `/me` API 调用增加 `?tenant_id=` query param |
| `Sidebar.vue` | 同上 |
| `WorkPanel.vue` | 同上 |
| `user_serializer.py` | `get_current_company` + `get_current_workspace` 增加 `request.GET` query param fallback |

## Test: ✅ 1 passed, 0 failed

## 修复原理

Before: `/me` API 调用 `/api/user/{uid}/accounts/users/me/` — NO tenant_id context → `get_current_workspace` 返回旧公司 workspace → 项目列表为空

After: `/me` API 调用 `/api/user/{uid}/accounts/users/me/?tenant_id=<current>` → `get_current_workspace` 正确解析当前公司 workspace → 项目列表正确

## 与上一个修复的关系

上一个修复 (Navbar/Sidebar URL tenant 优先) + 本修复 (`/me` API tenant context 传播) = **完整的公司切换上下文一致性**
