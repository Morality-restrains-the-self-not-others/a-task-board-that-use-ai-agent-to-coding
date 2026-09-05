# 价值流增量 — 项目文件树与变动列表 Tab

- **日期**: 2026-08-22
- **设计**: `docs/superpowers/specs/2026-08-22-file-tree-changes-tab-design.md`

## 触发者

任务执行者（已打开任务详情、已选中可写层）。

## 增量步骤

1. 打开任务详情 → 评论执行细节 → 任务关联
2. 选中层节点 → 出现文件树 / 变动 Tab（默认文件变动）
3. 浏览仓库树或切换到「文件变动」查看 diff 列表、预览、提交
4. 切回文件树时展开状态仍在

## 价值

同一视口内对照「全量树」与「本层变动」，减少纵向滚动。

## 测试点映射

见意图 `task_detail_file_tree_changes_tab.test-intent.md` T1–T8；注册到 `docs/flows/value-stream-test-integration.wsd` 任务详情段落。

## 跨流依赖

无。不改变克隆 / 提交 / 容器启动价值流，只改变展示编排。
