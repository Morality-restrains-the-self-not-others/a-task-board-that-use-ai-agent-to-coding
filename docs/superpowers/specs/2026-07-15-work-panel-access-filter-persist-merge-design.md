# 设计：工作面板 access 过滤持久化 + 筛选模态合并

- 日期：2026-07-15
- 状态：已采纳（goal-mode 自动决策）
- 前置：`2026-07-15-work-panel-access-filter-design.md`（009）
- `python_api_approval`: scoped-down（零新增 Python 接口；扩展既有 Go payload）

## 1. 问题

1. Header 人/小组过滤仅会话内有效，刷新丢失。
2. 旧「筛选」模态有 `filterOptions.assignee` 数据字段但无 UI；与 009 Header 控件重复语义，易双轨。

## 2. 成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | `access_filter` 写入 `work-panel-filters` payload | 同用户×工作空间 GET 回显 |
| S2 | 刷新/重进 work-panel 恢复人/小组选中 | chip 与看板过滤一致 |
| S3 | 切换工作空间加载该空间偏好 | 互不串扰 |
| S4 | 筛选模态含人/小组选择，与 Header **同一 accessFilter 状态** | 一处改、两处显 |
| S5 | 移除孤立 `filterOptions.assignee` 服务端查询路径 | 不再传 `?assignee=` |
| S6 | Go + 前端测例；Swagger 更新 | 通过 |

## 3. 方案（采纳 A）

| 方案 | 说明 |
|------|------|
| **A（采纳）** | payload 增 `access_filter: {kind,id,label}\|null`；version≥2；不存 memberIds（加载时重算） |
| B | 独立 API | 拒绝：已有 filters 端点 |
| C | 模态另做 assignee 下拉写 filterOptions | 拒绝：与 Header 双轨 |

## 4. Payload

```json
{
  "version": 2,
  "deliverable_filter_bars": [...],
  "access_filter": null
}
```

或：

```json
"access_filter": { "kind": "person"|"group", "id": "…", "label": "…" }
```

规范化：非法 kind/空 id → null；PUT 省略字段视为 null（清除）。

## 5. 模态合并

- 模态增加「人 / 小组」区块，选项与 Header 同源（accessPeople/accessGroups）。
- 选择调用同一 `selectAccessSubject`；清除与 chip 一致。
- `filterOptions` 仅保留 `search` + `priority`；删除 `assignee`。
- 重置：清除 search/priority **且** `clearAccessFilter`。

## 6. 架构

扩展既有 Rel_Flow（同 path PUT body 字段），不新建服务/表 → **可不新开架构版本**（与 006 同端点演进）；在 VERSION_HISTORY 记一条「payload 字段扩展」说明即可（可选）。本迭代文档声明无新组件。
