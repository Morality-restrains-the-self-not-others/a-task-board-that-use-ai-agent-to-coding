# 测试意图：Chrome 插件用户可见名称更名为「云端Coding: 自动创新助手」

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_display_name.intent.md`

## 测试目标

锁定品牌 SSOT、manifest、用户可见文案与 DevTools 面板创建参数，防止旧名「云端 Coding」回潮。

## 测试分层

- 单元/契约：`taskChromePlugin/test/plugin-brand.test.js`
- 回归：`taskChromePlugin/test/user-guide.test.js`（章节与 USER_GUIDE.md 同步）

## 用例矩阵

| ID | 场景 | 层级 | 期望 |
|----|------|------|------|
| T1 | plugin-brand.js | 单元 | `PLUGIN_DISPLAY_NAME` 为「云端Coding: 自动创新助手」 |
| T2 | manifest.json | 契约 | `name` 与 `default_title` 等于 SSOT |
| T3 | 用户可见文件 | 契约 | 不含「云端 Coding」，均含新显示名 |
| T4 | devtools.js | 契约 | `panels.create(PLUGIN_DISPLAY_NAME, …)` |
| T5 | content_scripts | 契约 | `plugin-brand.js` 在 `content.js` 之前 |
| T6 | user-guide | 回归 | 章节 id 仍在 USER_GUIDE.md |

## 数据与环境

- 不连真实浏览器；读源文件与 require SSOT 模块。

## 通过标准

```bash
cd taskChromePlugin && node --test test/plugin-brand.test.js test/user-guide.test.js
```

T1–T6 全部通过。

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | 初版 | 与功能意图同步 |
