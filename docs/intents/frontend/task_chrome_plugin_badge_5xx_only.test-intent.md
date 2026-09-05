# task_chrome_plugin_badge_5xx_only（测试意图）

## 测例对照

| ID | 场景 | 层级 | 期望 |
|----|------|------|------|
| T1 | `isHttp5xx(503)` | 单元 | true |
| T2 | `isHttp5xx(404)` / `0` / `200` | 单元 | false |
| T3 | `filterBadgeCountableRequests` 混合列表 | 单元 | 仅保留非 canceled 的 5xx |
| T4 | （手动）页面产生 404 与 503 | 手工 | 角标只因 503 递增；批量建任务只含 503 |

## 自动化入口

```bash
cd taskChromePlugin && npm test
```

覆盖：`test/capture-status.test.js` 中 `isHttp5xx`、`filterBadgeCountableRequests`。
