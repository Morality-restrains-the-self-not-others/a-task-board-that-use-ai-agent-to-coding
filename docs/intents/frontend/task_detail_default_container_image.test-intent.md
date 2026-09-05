# 测试意图：任务详情「关联项目」展示默认镜像

## 测试目标

验证关联项目标题栏只读展示任务默认容器镜像，解析顺序为：已安装镜像目录 → 任务嵌套 `container_image` → 评论 mention 标签 → 「已绑定」/「未设置」。

## 测试分层

| 层 | 范围 | 位置 |
|----|------|------|
| 单元 | 标签格式与解析回退 | `taskFE/app/src/utils/installedImageLabel.test.js` |
| 组件 | Toolbar / Panel 渲染 | `TaskDetailLinkedProjectsToolbar.test.js`、`TaskDetailLinkedProjectsPanel.test.js` |

无新后端契约，不做基础设施测。公网硬刷新验收见页面元素调整（非本文件门禁）。

## 用例矩阵

| ID | 场景 | 输入 | 期望 |
|----|------|------|------|
| U1 | 目录命中 | id + `{name, version}` | `name:version` |
| U2 | 名称已含版本 | name=`a:v1`, version=`v1` | `a:v1`（不重复） |
| U3 | 评论 mention 回退 | 目录无该 id，评论 `container_image_label` | mention 名 |
| U4 | 仅有 id | 无目录、无 mention | 展示「已绑定」 |
| U5 | 未设置 | 空 id | 展示「未设置」 |
| U6 | Toolbar | 传入 label | `data-testid="task-default-container-image"` 可见且含「默认镜像」 |
| U7 | Panel 只读 | 非编辑态 | 有默认镜像文案，无镜像 `<select>` |

## 数据与环境

- jsdom + Vue Test Utils / vitest（与现有 task-detail 组件测一致）。
- 不请求真实 `/api/cloud/installed-images`；目录由测试夹具传入。

## 通过标准

- 上表 U1–U7 全部绿。
- 评论 composer 既有「将使用镜像 …」测例仍绿（共用 `formatInstalledImageRunLabel`）。

## 业务意图 → 事件对照（测试侧）

| 业务意图 | 事件断言 | 例外理由 |
|---------|---------|---------|
| 关联项目区展示默认镜像 | 无 | 无对应事件：纯前端只读展示 |
