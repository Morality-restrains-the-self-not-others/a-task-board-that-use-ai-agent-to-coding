# Review: zTree 推送后 PR 按钮跳转审查页

- 日期: 2026-07-13
- 对照意图: `docs/intents/frontend/ztree_push_and_create_pr.*`
- 结果: **PASS**

## 对照

| 项 | 状态 |
|----|------|
| 推送成功不自动 open | ✅ |
| 写入 `pr_html_url` | ✅ |
| zTree 显示 `layer-ztree-pr-btn` | ✅ |
| 点击打开审查页 | ✅ |
| 刷新保留 URL（ahead=0） | ✅ |
| vitest | ✅ 25 passed |

## 非阻塞

- 全页刷新后若服务端层图不含 `pr_html_url`，按钮会消失（同会话内刷新已合并保留）。
- 未新增站内 PR 审查路由；目标为 GitHub `html_url`。
