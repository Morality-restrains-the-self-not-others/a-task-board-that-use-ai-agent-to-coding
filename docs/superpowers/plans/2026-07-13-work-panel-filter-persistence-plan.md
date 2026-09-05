# 工作面板过滤选项持久化 — 实施计划

- 日期：2026-07-13
- 设计：`docs/superpowers/specs/2026-07-13-work-panel-filter-persistence-design.md`

## Task checklist

- [x] **T1** 意图文档：`006_工作面板过滤选项持久化.intent.md` + `.test-intent.md`
- [x] **T2** Go migration：表 `user_workspace_work_panel_filters` + `table_ownership.yaml`
- [x] **T3** 领域 normalize + repository（Go）
- [x] **T4** GET/PUT handler + workspaces 子路由 + OpenAPI/Swagger
- [x] **T5** Go 测试红→绿
- [x] **T6** 前端 `workPanelFilterPersistence.js` + 单测
- [x] **T7** `WorkPanel.vue`：init GET、watch PUT、切空间加载
- [x] **T8** 网关 docs 启用；价值流/日志审计
- [x] **T9** Review + Ship（PR）

## 验证命令

```bash
cd taskProjectService && go test ./src/ -count=1 -run WorkPanelFilter
cd task2app/front_project/app && npm test -- --run workPanelFilterPersistence
```

## 日志要求（Step 8）

- GET/PUT：info，含 tenant_id、workspace_id、user_id、bars_count、ok/error
- 禁止记录完整敏感 token（本 payload 无 token）
