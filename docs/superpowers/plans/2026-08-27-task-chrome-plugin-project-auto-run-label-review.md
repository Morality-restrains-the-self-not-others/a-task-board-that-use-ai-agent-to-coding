# Review：Chrome 插件项目列表自动运行标注

- **Date:** 2026-08-27
- **Verdict:** pass（无 Critical / Required）

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | `default_auto_run === true` 才标可自动运行；缺字段安全默认为不可。浮窗与面板共用纯函数。 |
| Readability | SSOT 在 `lib/project-auto-run-label.js`。 |
| Architecture | 无新 API/事件；只读展示既有 GET 字段。 |
| Security | 项目名经调用方 `esc`/`escHtml`；徽章文案固定中文。 |
| Performance | 不新增请求。 |

## 安全审计

- 无密钥；无新写路径；XSS 转义已测。

## Intent → Event

只读展示，意图文档已书面无事件例外。

## CRG

`code-review-graph update --brief` 已跑；本增量无后端符号爆炸半径。

## 测试

- 新增 `test/project-auto-run-label.test.js`：T1–T9 全绿
- `node --test test/*.test.js`：447 pass / 0 fail
- e2e Playwright 10 fail：环境无 9222 Chrome，与本增量无关

## Nit（不修）

- `content.js` 仍超 500 行（OPT-20260827-032）
- 未按标注禁用「是否自动运行」勾选（OPT-20260827-033）
