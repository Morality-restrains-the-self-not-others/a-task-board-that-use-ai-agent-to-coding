# Review：Chrome 插件项目单选 + 自动运行联动

- **Date:** 2026-08-27
- **Plan:** `docs/superpowers/plans/2026-08-27-task-chrome-plugin-project-single-select-auto-run-plan.md`

## 对照计划

| Task | 结果 |
|------|------|
| T1–T3 resolve / pick / apply 单测 | 通过 |
| T4 源码契约 radio + 无全选 | 通过 |
| T5–T7 lib + 浮窗 + 面板 | 已落地 |
| T8 使用说明 + manifest 1.8.16 | 已落地 |
| T9 `npm test` | 457 通过 |

## 安全 / 权限

无新 API。自动运行禁用为 UX 钳制；创建仍走既有后端门禁。

## Intent → Event

书面例外：纯 UI，无新 MQ 事件。

## 残留

- `content/content.js` 仍 >500 行（OPT-20260827-032）
- 未对齐工作面板镜像+硬件模版完整启机门禁（有意裁剪）
