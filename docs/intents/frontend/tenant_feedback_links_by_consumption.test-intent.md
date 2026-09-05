# 测试意图 — 意见与建议链接（前端）

- **对应意图:** `tenant_feedback_links_by_consumption.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| F1 | GET 返回两组 | 子菜单两个组名段，链接 href 与 API 一致 |
| F2 | GET 空数组 | 一级菜单仍在，子菜单空态 |
| F3 | 链接点击 | 原生 `<a target="_blank">`，无 prevent+router.push |
| F4 | 无 feedback:view | 无「意见与建议」 |
| F5 | 超管保存 | 单次点击只发一次 POST/PUT，带 Idempotency-Key |
| F6 | 缩窄侧栏 | 点击一级后展开再显示子菜单 |
