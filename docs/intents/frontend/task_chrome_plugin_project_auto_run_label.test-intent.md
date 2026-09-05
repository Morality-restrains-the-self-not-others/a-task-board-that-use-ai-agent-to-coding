# 测试意图：Chrome 插件项目列表标注是否可自动运行

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_project_auto_run_label.intent.md`

## 测试目标

锁定项目列表标注只读 `server_run_template.default_auto_run`，浮窗与 DevTools 共用纯函数，且 HTML 转义项目名。

## 测试分层

- 单元：`taskChromePlugin/test/project-auto-run-label.test.js`
- 源码契约：同文件扫描 `content.js` / `workspace.js` / `manifest.json` / `panel.html` / 使用说明

## 用例矩阵

| ID | 场景 | 层级 | 期望 |
|----|------|------|------|
| T1 | `default_auto_run: true` | 单元 | `projectAllowsAutoRun` true，徽章「可自动运行」 |
| T2 | `default_auto_run: false` | 单元 | false，「不可自动运行」 |
| T3 | 无 `server_run_template` | 单元 | false，「不可自动运行」 |
| T4 | 模版为数组 / 非对象 | 单元 | false |
| T5 | 名称含 `<script>` | 单元 | caption HTML 转义，不含裸 `<script>` |
| T6 | `formatProjectListLabel` | 单元 | `名称（可自动运行）` / `名称（不可自动运行）` |
| T7 | 浮窗与面板渲染调用该模块 | 源码 | `content.js` 与 `workspace.js` 含 `ProjectAutoRunLabel` |
| T8 | 注入顺序 | 源码 | `manifest.json` 与 `panel.html` 在 content/workspace 之前加载该 lib |
| T9 | 使用说明 | 源码 | `USER_GUIDE.md` 与 `user-guide.js` 含「可自动运行」 |

## 数据与环境

- 不连真实网关；纯函数测例构造项目对象。

## 通过标准

```bash
cd taskChromePlugin && node --test test/project-auto-run-label.test.js test/user-guide.test.js
```

全部 T1–T9 通过。
