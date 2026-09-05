# Ship Report: 公司切换后页面刷新回退 — 修复

> Date: 2026-06-28
> Pipeline: 1→3→4→5→6→7→8→9 (DDD skipped — no new domain concepts)

## 交付成果

### 已修改文件

| 文件 | 改动 | 说明 |
|------|------|------|
| `task2app/front_project/app/src/components/Navbar.logic.vue` | +4 行 | `applyMePayload`: URL tenant 优先于 API `current_company` |
| `task2app/front_project/app/src/components/Sidebar.vue` | +11 行 | `initData`: URL tenant 优先于 API `current_company` |
| `task2app/Saas_project/accounts/serializers/user_serializer.py` | +9 行 | `get_current_company`: 增加 `tenant_id` URL 感知 |
| `conf/value-stream.yaml` | +8 行 | 新增 `company-switch-url-context` 步骤 (planned) |

### 已产出文档

| 文档 | 路径 |
|------|------|
| 设计文档 | `docs/specs/company-switch-revert-on-refresh-design.md` |
| 价值流 | `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-value-stream.md` |
| NFR 澄清 | `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-nfr-clarification.md` |
| DDD (跳过) | `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-ddd.md` |
| 实施计划 | `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-plan.md` |
| 代码审查 | `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-review.md` |

### 测试结果

- ✅ `UserViewSet_test.py` — 1 passed, 0 failed

## 修复原理

**Before**: `currentTenant` = API `current_company` (always `.first()` in DB) → 切换后回退

**After**: `currentTenant` = URL `route.params.tenant` (if valid) → API fallback → 与 URL 一致

## Learnings

1. **URL 是真源** — `/tenant/:tenant` 已经是当前公司的权威标识，组件状态应以此为优先，API 响应仅作 fallback
2. **模式一致性** — `WorkPanel.vue` 早已正确使用 URL tenant，Navbar/Sidebar 是遗漏的缺陷
3. **`get_current_workspace` 已有 URL 感知** — 同文件中的 workspace 方法做了正确示范，`get_current_company` 是唯一的遗漏
