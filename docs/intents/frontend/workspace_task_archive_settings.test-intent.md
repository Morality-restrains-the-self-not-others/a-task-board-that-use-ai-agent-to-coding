# 测试意图：工作空间「套餐设置」打开任务存档档位

## 测试目标

验证 settings/task-panel「套餐设置」点击打开存档档位模态，列表展示档位文案，保存走既有 PATCH。

## 测试分层

| 层 | 文件 | 覆盖 |
|----|------|------|
| 组件 | `WorkspaceSettingsTaskPanelActions.unit.test.js` | 套餐设置按钮存在并 emit `archive` |
| 组件 | `WorkspaceSettingsTaskPanel.archive-modal.test.js` | 点击打开模态；行展示档位；保存 PATCH + Idempotency-Key |
| E2E | 浏览器手测公网 task-panel | 点击「套餐设置」出现模态、无错误弹层 |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 工作空间列表已加载 | 点「套餐设置」 | 出现「套餐设置 · 任务存档时间」模态 |
| T2 | `task_archive_tier=7d` | 渲染列表 | 行文案含「存档：免费存放期(7天)」，不含「该功能暂未开放」 |
| T3 | 模态已打开 | 选档位并保存 | PATCH workspace，body `task_archive_tier`，头 `Idempotency-Key` |
| T4 | 操作条 | 渲染 | `[data-testid=open-task-archive-settings]` 可见且文案为套餐设置 |

## 数据与环境

- Vitest + jsdom；`apiFetch` mock 工作空间列表与 PATCH。
- 不 stub `WorkspaceSettingsTaskPanelActions`（T1/T2/T3 要点击真实按钮）。

## 通过标准

T1–T4 全绿；既有 click-guard / default-badge 测例仍通过。
