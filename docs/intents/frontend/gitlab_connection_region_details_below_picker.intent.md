# 功能意图：GitLab 连接页区域详情在下拉之下

## 背景与目标

租户设置「GitLab」页 `data-testid="gitlab-region-picker"` 与系统内建 GitLab 详情 `data-testid="gitlab-builtin-resources"` 目前互斥：未购买只显示区域下拉；一旦所选区域 `provisioning_status` 为 `active` / `pending_admin`，下拉整块被详情替代。用户无法再切换区域查看其他区域的配额与入口。

目标：区域下拉始终保留；所选区域的 GitLab 详情放在下拉选择器下方；切换选项只更新下方详情，不替换选择器。

## 范围与边界

- 范围内：`WorkspaceSettingsGitlabConnection.vue` 布局与可见性；对应单测；本意图与测试意图；价值流测试点
- 范围外：购买/开通 API、自建 GitLab OAuth 区块、导航栏「代码仓库」下拉、区域 CRUD

## 约束与风险

- `data-testid="gitlab-region-picker"` / `gitlab-region-select` / `gitlab-builtin-resources` 保持不变
- 无默认区域：未选区域时不伪造详情；空态仍显示「请选择区域」
- 纯前端展示布局，无服务端状态变更
- 切换区域仍走既有 `watch(region) → load()`（`GET .../gitlab-resources/?region=`），禁止后台轮询

## 验收标准

1. 无论所选区域是否已购买/已开通，`gitlab-region-picker` 与其中的 `gitlab-region-select` 始终可见
2. 选中某一区域后，`gitlab-builtin-resources` 出现在同一 picker 区块内、`<select>` 之后（DOM 顺序：select 在前、详情在后）
3. 将下拉从区域 A 改为区域 B 后，详情展示 B 的名称（来自选项或 `regionName`），picker 仍在
4. 未选择区域且 `provisioningStatus=not_purchased` 时，不展示 `gitlab-builtin-resources`（空态仅下拉）
5. 已开通（`active` / `pending_admin`）时 picker 不被隐藏；等待开通文案仍在详情内

## 实施计划

1. 去掉 picker 上「仅未购买才显示」的 `v-if`
2. 将内建 GitLab 详情移入 picker 区块、放在 `<select>` 下方；可见性改为「已选区域或已有资源」
3. 详情标题/链接优先用当前下拉选项的 `name` / `gitlab_web_url`，切换时立即反映选项
4. 单测覆盖共存、DOM 顺序、切换选项

## 变更记录

- 2026-08-22：相对上一版（picker 与详情互斥），改为 picker 常驻、详情随选项更新且位于下拉之下。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 区域下拉下方展示对应 GitLab 详情 | — | — | 前端只读渲染 | — | 纯前端展示布局，无服务端状态变更 |
