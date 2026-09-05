# 测试意图：项目文件树与变动列表 Tab

对应功能意图：`task_detail_file_tree_changes_tab.intent.md`

## 范围

- 组件：`TaskDetailLayerFilesTabs.vue`、`TaskDetailTaskLayerAssociationPanel.vue`
- 单元：`TaskDetailLayerFilesTabs.test.js`、`TaskDetailTaskLayerAssociationPanel.layer-files-tabs.test.js`
- E2E：既有 layer-changes / project-file-tree Playwright 在切 Tab 后仍能操作对应面板；新增 Tab 切换用例

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 选中层且文件树 layerId 就绪 | 出现 `layer-files-tablist`；默认 `aria-selected` 为文件变动；变动面板可见、文件树面板 `display:none` 但仍挂载 |
| T2 | 点击「项目文件树」Tab | 文件树面板可见；变动面板隐藏但节点仍在 DOM |
| T3 | 再点回「文件变动」 | 变动面板重新可见 |
| T4 | `displayCount>0` | 变动 Tab 文案含 `· {N}` |
| T5 | 切换 `layerId` | 活动 Tab 复位为文件变动 |
| T6 | 无 layerId | 不渲染 Tab 栏，保持旧的空选中态 |
| T7 | Playwright：文件变动刷新仍触发文件树重拉 | 默认变动 Tab 下点刷新，文件树 API 计数增加 |
| T8 | Playwright：默认可见文件变动面板 | 选中层后 `文件变动` Tab 选中且变动面板可见；点「项目文件树」后树 body 可见 |

## 不测

- 变动列表提交/暂存/续拉的业务语义（已有独立意图与测例）
- 文件树懒加载与预览（已有独立测例）
