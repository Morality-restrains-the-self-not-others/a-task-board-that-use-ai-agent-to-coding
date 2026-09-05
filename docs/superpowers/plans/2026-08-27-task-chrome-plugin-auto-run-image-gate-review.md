# Review：Chrome 插件自动运行与已安装镜像挂钩

- **Date:** 2026-08-27
- **Plan:** `docs/superpowers/plans/2026-08-27-task-chrome-plugin-auto-run-image-gate-plan.md`

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | 项目允许但无镜像 → 自动运行禁用；提交 `auto_run` 无镜像拦截。批量创建原先过滤错误文案会吞镜像门禁，已改为展示全部 `validateCreateTaskForm` 结果。 |
| Readability | 纯函数集中在 `project-auto-run-label.js`，与 Git 身份门禁同构。 |
| Architecture | 无新服务/API。意图文档书面无 MQ 事件例外。 |
| Security | 无新权限面；纯前端校验不伪造 `data-traceId`。 |
| Performance | 无新网络请求；仅 DOM change 重算。 |

## 安全审计（裁剪）

- [x] 无密钥
- [x] 无新用户输入进 SQL
- [x] 创建失败展示为纯前端校验时无 `data-traceId`
- n/a CORS/CSP/后端鉴权变更

## Log Audit

无新出站 HTTP。拦截路径走既有 `showResult` / `showR`。Content script 无 Loki 结构化日志（与既有插件一致）。

## Intent→Event

书面例外：纯 UI。创建任务成功路径未改。

## CRG

`code-review-graph update --brief` 已在流水线开始执行。变更符号限于 taskChromePlugin JS。

## Critical / Required

无阻断项。硬件运行模版前端门禁仍为 OPT-20260827-034 余项。

## 测试

`cd taskChromePlugin && npm test` — 472 passed / 0 failed。
