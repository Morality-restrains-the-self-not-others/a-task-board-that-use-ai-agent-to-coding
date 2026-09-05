# 功能意图：用户列表表头列过滤器

- **页面**: `/system-admin/users/`
- **元素**: `div#users` 用户表格 `thead` 第二行

## 意图

在每一列标题下方提供过滤器，修改后刷新当前 tab 的用户列表。

## 行为

1. 活跃/已归档 tab 的表格 `thead` 在标题行下渲染过滤行（`data-testid="user-list-column-filters"`）。
2. 文本列可输入；枚举列下拉；日期列起止日期；操作列为「重置」。
3. 变更后请求 `GET /api/system-admin/users/` 携带对应 query，offset 回到 0。
4. 重置清空列过滤并重新请求（保留 tab 的 is_archived 与顶部 q）。
5. 推荐码申请 tab 不显示该过滤行。
6. Anti-Replay-OK: 只读 GET；文本防抖 400ms。

## 非目标

- 推荐码申请列表过滤。
- 去掉顶部全局搜索框。

## 业务意图 → 事件对照

**无对应新事件（书面例外）**：前端只读查询。
