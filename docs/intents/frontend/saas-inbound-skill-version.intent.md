# 功能意图：容器→SaaS 接口版本（厂商门户）

- **日期**: 2026-08-20
- **状态**: 已实施
- **页面**: `/saas-machine-container`、厂商门户镜像添加/编辑

## 背景与目标

厂商查看「容器→SaaS 接口」时能看到契约版本；添加/编辑容器镜像版本时必须选择该契约版本，而不是只填镜像标签。

## 验收标准

1. skill 页展示当前接口版本（如 `v1`），`data-testid="saas-inbound-skill-version-badge"`。
2. 版本选择器可切换已发布文档；未知版本显示错误且带 `data-traceId`（若来自 HTTP）。
3. 添加/编辑镜像模态有必选「容器→SaaS 接口版本」下拉，`data-testid="saas-inbound-skill-version-select"`；未选不得提交。
4. 下拉选项来自 `GET /api/ai-provider/saas-inbound-skill-versions/`，展示 `v{n}` + summary。
5. 导航仍为真实 `router-link`（`data-testid="nav-saas-machine-container-skill"`），禁止 `@click.prevent`。
6. 保存草稿按钮有同步门闩 + `Idempotency-Key`（写路径）。

## 业务意图 → 事件对照

| 业务意图 | 事件 | 例外理由 |
|----------|------|----------|
| 浏览 skill / 选版本查看 | — | 纯查询 |
| 保存镜像并选定接口版本 | `ContainerImageSaasInboundSkillVersionAssigned` | 由 taskAiProvider 发布 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-08-20 | 初版：契约版本展示 + 镜像表单必选 |
| 2026-09-01 | 厂商弹窗不再因 catalog 500 显示 skill version catalog unavailable（后端 embed） |
