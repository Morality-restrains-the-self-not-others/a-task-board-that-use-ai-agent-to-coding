# 角色权限分析：创建任务可选字段显隐设置

**日期**: 2026-07-18  
**设计**: `2026-07-18-create-task-field-settings-design.md`

## 改动点权限矩阵

| 能力 | 角色 | 判定 |
|------|------|------|
| GET create-task-field-settings | 租户内已认证用户（可访问该工作区） | 与 task-kind-options GET 一致：gateway user + workspace∈tenant |
| PUT create-task-field-settings | 同上 | 本期不新增独立「仅管理员可写」门禁（与 code-lang-options PUT 对齐）；后续若收紧 options 写权限则一并收紧 |
| 创建任务表单按配置显隐 | 任意可打开 work-panel 创建的用户 | 只读消费配置；无法绕过 settings 写入（前端隐藏；服务端创建任务仍接受字段，属既有行为） |

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| 成员改配置影响全员创建体验 | 与现有「任务类型/编程语言」配置同信任模型；UI 仅在 settings/task-panel |
| 隐藏后客户端仍可手工 POST 字段 | 可接受；本期目标是 UX 精简，非安全硬门禁 |

## 结论

无新增独立权限模型；复用工作区配置读写边界。无需改 RBAC 表。
