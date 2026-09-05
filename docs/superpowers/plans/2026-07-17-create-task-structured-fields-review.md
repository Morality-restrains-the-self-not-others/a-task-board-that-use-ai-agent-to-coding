# Review：创建任务结构化可选字段

**日期**: 2026-07-17  
**结论**: 通过（无 critical）

## 检查项

| 项 | 结果 |
|----|------|
| 成功标准：7 可选字段 UI | ✅ CreateTaskBasicFields |
| compose 入 description / 防重复 | ✅ applyComposedDescriptionToTask |
| 编辑回填 | ✅ hydrateStructuredFieldsFromDescription |
| 无新 API/权限面 | ✅ |
| 单测 | ✅ 29 passed（compose + CreateTaskModal） |
| 公网 SPA collectstatic | ✅ runall-lifecycle.sh build；manifest md5 一致 |
| Log / Intent→Event | n/a（前端拼装，无新 MQ） |

## 附带入库说明

本地工作区已存在未提交的 CreateTaskModal 拆分（子组件 + composables）。本次 PR 一并纳入，否则仅改 BasicFields 无法在干净 checkout 构建。

## Important（非阻塞）

- Chrome 插件创建任务字段未对齐 → OPT
- 任务详情页仍以整段 Markdown 编辑，无独立结构化控件 → OPT
