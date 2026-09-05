# 测试意图：工作空间设为默认后 badge 与唯一默认

对应：`docs/intents/frontend/workspace_set_default_takes_effect.intent.md`

## 测试目标

验证「是否设为默认」保存后列表 badge 跟 `is_default`，且同一租户只有一个默认工作空间。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 组件（Vue） | badge 在 is_default 行；勾选后 POST `is_default:true` |
| Go | 更新/创建设默认时清除同租户兄弟；不碰其它租户 |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 列表 A `is_current=true,is_default=false`，B 相反 | 渲染工作空间管理 | 仅 B 有 `workspace-default-badge` |
| T2 | 打开添加弹窗并勾选「是否设为默认」 | 保存 | POST body `is_default===true` |
| T3 | 租户已有默认 A | PUT B `is_default:true` | B 为默认且 A 不再是 |
| T4 | 租户已有默认 A | POST 新空间 `is_default:true` | 新空间为默认且 A 不再是 |
| T5 | 租户 t2 已有默认 | 在 t1 切换默认 | t2 默认不变 |

## 可执行测试

- `taskFE/app/src/views/WorkspaceSettingsTaskPanel.default-badge.test.js`
- `taskProjectService/src/workspace_default_test.go`
