# 权限/价值流/NFR/领域/计划（合并简报）

- 日期：2026-07-15
- 设计：`2026-07-15-work-panel-access-filter-persist-merge-design.md`

## 权限
| 改动点 | 结论 |
|--------|------|
| PUT/GET work-panel-filters 扩字段 | 仍仅当前用户自己的行；无 IDOR 新面 |
| 模态人/小组 | 选项仍来自 workspace-permissions |

## 价值流
选人/组 → debounce PUT payload → 刷新 GET → hydrate memberIds → 看板过滤。

## NFR
L2；PUT 失败不打断 UI；memberIds 不落库（组成员变动后下次 hydrate 更新）。

## 领域
AccessFilterPref 值对象 `{kind,id,label}`；无新 MQ 事件（UI 偏好例外）。

## 计划
1. Go normalize + OpenAPI + test
2. 前端 persistence + hydrate
3. 模态合并 + 去 assignee
4. 测例 + SPA build + PR
