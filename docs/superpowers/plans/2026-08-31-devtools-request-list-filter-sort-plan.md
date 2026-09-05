# 实施计划：DevTools 请求列表过滤与排序

- **Date:** 2026-08-31

## Tasks

- [ ] **T1** 红：`test/request-list-query.test.js` — filter search/method/status/type、sort 各列、nextSortState、limit、稳定 id
- [ ] **T2** 绿：`lib/request-list-query.js` 使 T1 通过
- [ ] **T3** 面板接线：`panel.html` 类型芯片 + 可排序表头；`panel.css`；`single-request.js` 调用纯函数；加载 script
- [ ] **T4** 契约测试：panel.html 含 sort buttons / type chips；user-guide 文案
- [ ] **T5** 意图文档 + INDEX F-116
- [ ] **T6** bump `manifest.json` version（用户可见）
- [ ] **T7** 跑 `node --test` 相关文件 + 既有 panel-request 单测

每切片保持可编译；无 MQ 任务（纯前端例外）。
