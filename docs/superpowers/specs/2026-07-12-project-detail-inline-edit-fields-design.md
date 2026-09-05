# 项目详情页：标签 / 默认自动运行 / 已安装镜像 点击即编辑

日期：2026-07-12  
状态：已批准（goal-mode 自动采用）

## 问题

用户访问项目详情页时，「项目标签」「默认自动运行」「已安装镜像」均为只读；要改须点「编辑项目」进整页表单，路径过长。

## 目标

详情页上述三项 **点击展示区即可进入字段级编辑**，保存后就地刷新，无需跳转编辑页。

## 非目标

- 不新增后端 API；沿用 `PATCH /api/tenant/{t}/projects/{id}/`
- 不改架构拓扑（无新服务）
- 不在本次把名称/描述/Git 等也做成 inline（可后续扩展）
- 不移除「编辑项目」整页入口

## 方案（采用）

字段级 inline edit，抽成 `ProjectDetailInlineEditableFields.vue`（`ProjectDetail.vue` 已超 500 行门禁）。

| 字段 | 展示 | 点击后 | 保存 |
|------|------|--------|------|
| 项目标签 | badge /「未设置」 | `ProjectTagsInput` + 保存/取消 | `PATCH { tags }` |
| 默认自动运行 | 已启用/未启用 | checkbox（门槛同编辑页） | `PATCH { server_run_template: {…existing, default_auto_run} }` |
| 已安装镜像 | 展示文案 | `<select>` + 保存/取消 | `PATCH { container_image_id, container_image }` |

### 门槛（与 ProjectEdit 一致）

启用默认自动运行须：已选镜像 **且** 已配置运行模版（`projectHasConfiguredRunTemplate`）。不满足时 checkbox 禁用并提示。清除镜像时若自动运行曾开启，一并写回 `default_auto_run: false`。

### 交互细节

- 同时仅一个字段处于编辑态
- 编辑区 `cursor-pointer` + hover 提示「点击编辑」
- 保存失败展示字段级错误，不吞错
- `data-testid`：`project-tags-display` / `project-default-auto-run-display` / `project-container-image-display`（保留）及对应 `-edit` / `-save`

## 架构影响

无。纯前端 UX；不更新 `docs/architecture/`。

## 测试

- Vitest：纯函数 patch body + 组件点击进入编辑、保存调用 PATCH
- Playwright（可选增量）：详情页点击标签进入编辑控件可见
