# 意图：任务详情「关联项目」展示默认镜像

## 背景与目标

任务详情页「关联项目」标题栏（`justify-between`）右侧为空。任务级默认容器镜像存在于 `task.container_image_id`（创建任务时选择，评论 `@镜像` 可改写），但 ServerConfig 镜像选择 UI 已迁到评论 composer，用户在关联项目区看不到当前默认镜像。

目标：在「关联项目」标题同行右侧只读展示默认镜像名称（含版本，若可得）。

## 范围与边界

- 范围内：taskFE `TaskDetailLinkedProjectsToolbar` 展示；从已安装镜像目录解析 `name:version`；目录未命中时回退评论 `installed_image` mention / `container_image_label`。
- 范围外：不在此区提供镜像下拉或 PATCH；不改任务 API、不水合 `container_image` 对象；不恢复已移除的全局「镜像运行状态」header。

## 约束与风险

- 纯前端只读展示，无服务端状态变更。
- 任务 API 的 `container_image` 恒为 `null`，必须以 `container_image_id` + 已安装镜像列表（及评论 mention）解析文案。
- 绑定 ID 已从租户目录卸载时，不得误显示目录里另一个同名镜像；可显示 mention 名或「已绑定」。

## 验收标准

1. 「关联项目」标题行右侧存在 `data-testid="task-default-container-image"`，文案含「默认镜像」。
2. 目录能解析该 ID 时展示 `name:version`（与创建任务下拉、评论 `@镜像` 标签格式一致）。
3. 目录未命中但评论 mention 同 ID 时展示 mention 名称（如 `trae-agent`）。
4. 有 `container_image_id` 但无法解析名称时展示「已绑定」。
5. 无默认镜像 ID 时展示「未设置」。
6. 只读，不含选择器 / 保存按钮。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 关联项目区展示默认镜像 | — | — | — | — | 无对应事件：纯前端只读展示，复用既有任务字段与已安装镜像列表 |

## 实施计划

1. 抽出 `formatInstalledImageRunLabel` / `resolveTaskDefaultImageLabel` 纯函数 + 单测。
2. ServerConfig `#after-mirror` 作用域插槽下发 `selectedImageId` 与 `installedImages`。
3. `TaskDetailLinkedProjectsToolbar` 标题栏右侧展示解析结果。

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-20 | 初版 | 任务详情关联项目标题栏需显示默认使用的镜像 |
