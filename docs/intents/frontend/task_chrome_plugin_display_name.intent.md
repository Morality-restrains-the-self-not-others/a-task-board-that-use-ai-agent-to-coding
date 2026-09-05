# 意图：Chrome 插件用户可见名称更名为「云端Coding: 自动创新助手」

## 背景与目标

- 背景：扩展商店、工具栏、弹窗、DevTools 面板与使用说明仍使用旧名「云端 Coding」（中间有空格），与产品对外品牌不一致。
- 目标：所有用户可见名称统一为精确字符串 `云端Coding: 自动创新助手`（云端与 Coding 之间无空格；冒号后接副标题）。内部目录名、日志前缀、`trae-service`、npm package 名保持 `taskChromePlugin` / `task-chrome-plugin`。

## 范围与边界

- 范围内：`manifest.json` 的 `name` / `action.default_title`、popup/panel/devtools 标题与文案、页内悬浮球 tooltip、`lib/user-guide.js`、`docs/USER_GUIDE.md`、README、品牌 SSOT `lib/plugin-brand.js`。
- 范围外：不改扩展 ID、权限、任务创建 API、OAuth、日志前缀。

## 约束与风险

- Chrome `name` 上限 45 字符；目标串约 16 字，可完整展示。
- 弹窗宽约 300px，长标题须允许换行，禁止横向溢出把状态徽标挤出。
- `plugin-brand.js` 必须在 `content.js` / `devtools.js` 之前注入，否则 `PLUGIN_DISPLAY_NAME` 未定义。

## 验收标准

1. `PLUGIN_DISPLAY_NAME === '云端Coding: 自动创新助手'`。
2. `manifest.name` 与 `action.default_title` 等于该 SSOT。
3. 用户可见源文件不再出现旧名「云端 Coding」。
4. DevTools `panels.create` 使用 `PLUGIN_DISPLAY_NAME`。
5. `node --test test/plugin-brand.test.js` 全绿。

## 实施计划

1. SSOT `lib/plugin-brand.js` + 契约测试先行。
2. 注入顺序与各入口文案对齐；version bump `1.8.12`。
3. 使用说明两份 SSOT 与 README 同步。

## 业务意图 → 事件对照

| 业务意图 | 领域事件 | MQ Topic | 说明 |
|----------|----------|----------|------|
| 更名用户可见品牌 | （无） | — | **无对应事件**：浏览器扩展本地展示名，不投递领域事件 |

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | 初版 | 产品要求插件命名为「云端Coding: 自动创新助手」 |
