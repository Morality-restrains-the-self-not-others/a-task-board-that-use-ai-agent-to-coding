# 角色权限分析 — autoRunStep.md

- **日期**: 2026-07-14
- **设计**: `2026-07-14-auto-run-steps-md-design.md`
- **goal-mode**: 自动产出

| 能力 | 角色 | 权限边界 |
|------|------|----------|
| 读 catalog / 开发中预览 | 租户已登录用户 | 开发中仅绑定厂商的 saas_user |
| 安装镜像（含开发中） | 租户用户 | 既有 installed-images 门禁 |
| 创建任务展示说明 | 任务创建者 | 读已安装镜像快照 |
| 详情 live API | 任务成员 | 经 CGW + 容器 token |
| 触发抽取 | 厂商门户 / 内部 | vendor CRUD；extract 为 internal secret |
| 改 autoRunStep.md | 镜像构建者 | 仅镜像文件系统，非 SaaS 写 |

无新增公网特权接口；extract 仅 internal。
