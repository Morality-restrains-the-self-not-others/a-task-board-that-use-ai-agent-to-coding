# 测试意图：DevTools 请求列表过滤与排序

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_devtools_request_filter_sort.intent.md`

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 | `request-list-query.js` 过滤/排序/切换 |
| 契约 | panel.html 表头与类型芯片；user-guide 文案 |
| 回归 | 既有 panel-request-bootstrap / e2e 列表引导 |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | search 命中 URL | 仅匹配行留下 |
| T2 | search canceled | 命中已取消请求 |
| T3 | method=POST | 非 POST 剔除 |
| T4 | status=4xx | 仅 400–499 |
| T5 | status=canceled | 仅 canceled |
| T6 | type=xhr | fetch/xhr/preflight 留下，script 剔除 |
| T7 | sort timestamp desc | 新请求在前（默认） |
| T8 | sort time asc | 耗时小的在前 |
| T9 | sort method asc | DELETE 在 GET 前（字典序） |
| T10 | sort status：canceled 视为 -1 | canceled 在 200 前（asc） |
| T11 | 同列 nextSortState | desc↔asc |
| T12 | 新列 nextSortState | 数值 desc / 字符串 asc |
| T13 | limit 100 | 只返回 100 条 |
| T14 | 使用说明 | 提及类型筛选与列头排序 |

## 数据与环境

```
cd taskChromePlugin && node --test test/request-list-query.test.js test/user-guide.test.js test/panel-request-bootstrap.test.js
```

## 通过标准

上表全部通过；无对应 MQ 事件。
