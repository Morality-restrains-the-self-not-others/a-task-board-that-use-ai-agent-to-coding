# DDD：Chrome 插件项目列表自动运行标注

- **Date:** 2026-08-27
- **Bounded context:** taskChromePlugin（展示适配器）读 taskProjectService 项目实体已有 VO

## 模型

不新增服务端聚合。前端值对象：

- **ProjectAutoRunAllowance**（VO）：由 `server_run_template.default_auto_run === true` 判定。
- **ProjectListCaption**（VO）：`displayName` + 允许/不允许徽章。

无仓储端口变更。无领域事件（只读查询例外，见意图对照表）。

## NFR → 模型

- XSS：caption 组装必须经转义函数，禁止拼接原始 `name`。
- 缺字段：VO 默认「不允许」，与后端 `AUTO_RUN_PROJECT_NOT_ALLOWED` 同向。
