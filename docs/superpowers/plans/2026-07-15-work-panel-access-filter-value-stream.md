# 价值流：工作面板按可访问人或小组过滤

- 日期：2026-07-15
- 设计：`docs/superpowers/specs/2026-07-15-work-panel-access-filter-design.md`

## 1. 用户价值

协作成员在工作面板快速只看「某人 / 某组」相关任务，减少在全量看板中扫视成本。

## 2. 端到端步骤

```
打开 work-panel
  → 加载工作空间任务 + collaborators + permissions
  → 用户打开「人/小组」过滤
  →（若小组）拉取 members 并映射 company_member_id
  → 客户端过滤 todos（owner∪assignees）
  → 看板重绘；chip 可清除
```

## 3. 最小可行增量（MVI）

| 增量 | 内容 | 验收 |
|------|------|------|
| MVI-1 | `workPanelAccessFilter` 纯函数 + 单元测 | T1–T7 |
| MVI-2 | Header UI + WorkPanel 接线 + 人过滤 | S1/S2/S4 |
| MVI-3 | 小组过滤 + members 映射 | S3 |
| MVI-4 | Playwright 冒烟 | T8–T10 |

## 4. 测试点映射

见 `task2app/docs/intents/frontend/work_panel/009_*.test-intent.md`。

## 5. 非本流

- 过滤持久化、服务端查询参数、权限角色过滤。
