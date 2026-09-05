# 测试意图：Chrome 插件自动运行与已安装镜像必选挂钩

## 对应功能意图

`task_chrome_plugin_auto_run_image_gate.intent.md`

## 测试点

| ID | 场景 | 期望 | 文件 |
|----|------|------|------|
| T1 | 允许自动运行的项目 + 无镜像 | `enabled=false`，hint 含「请先选择已安装镜像」 | `test/project-auto-run-label.test.js` |
| T2 | 允许自动运行的项目 + 有镜像 + preference true | `enabled=true, checked=true` | 同上 |
| T3 | 无项目时即使有镜像 | 仍提示「请先选择项目」 | 同上 |
| T4 | 项目不允许时即使有镜像 | 仍提示未允许自动运行 | 同上 |
| T5 | 项目允许时镜像标签必选外观 | `requiredMarkVisible=true`，空选项「请选择已安装镜像」 | 同上 |
| T6 | 项目不允许时镜像可选外观 | `requiredMarkVisible=false`，空选项「无」 | 同上 |
| T7 | `validateCreateTaskForm` auto_run 无镜像 | 返回「请先选择已安装镜像」 | `test/create-task-payload.test.js` |
| T8 | `validateCreateTaskForm` auto_run 关无镜像 | 空字符串（不因镜像拦截） | 同上 |
| T9 | 浮窗/面板把镜像 select 传入 resolve | content.js / workspace-projects.js 含 `hasInstalledImage` | `test/project-auto-run-label.test.js` |
| T10 | 使用说明写明须先选镜像 | USER_GUIDE + user-guide.js | `test/user-guide.test.js` |

## 业务意图 → 事件对照

与功能意图相同：无新 MQ 事件。
