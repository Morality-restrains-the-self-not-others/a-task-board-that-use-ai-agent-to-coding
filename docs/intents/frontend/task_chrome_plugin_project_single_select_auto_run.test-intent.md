# 测试意图：Chrome 插件项目单选 + 自动运行联动

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_project_single_select_auto_run.intent.md`

## 测试目标

锁定：项目单选；自动运行控件状态由 `projectAllowsAutoRun` + 勾选偏好推导；浮窗与面板接入同一纯函数。

## 测试分层

- 单元：`taskChromePlugin/test/project-auto-run-label.test.js`
- 源码契约：扫描 `content.js` / `workspace.js` / `panel.html` / 使用说明

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T10 | 无选中项目 | enabled=false, checked=false, hint 含「请先选择项目」 |
| T11 | 项目 default_auto_run 非 true | enabled=false, checked=false, hint 含「未允许自动运行」 |
| T12 | 允许且 preference true | enabled=true, checked=true |
| T13 | 允许且 preference false | enabled=true, checked=false |
| T14 | pickSingleProjectId | 多候选取第一个仍在 available 中的 id |
| T15 | applyAutoRunControlToElements | disabled/checked/hint 写入 DOM |
| T16 | 浮窗/面板 | radio + project-radio，无 select-all，调用 resolveAutoRunControlState |
| T17 | 使用说明 | 不再写项目「可多选」；写明自动运行随项目能力 |

## 通过标准

```bash
cd taskChromePlugin && node --test test/project-auto-run-label.test.js test/user-guide.test.js
```
