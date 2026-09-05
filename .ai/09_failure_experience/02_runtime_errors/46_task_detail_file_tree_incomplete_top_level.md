# [运行时] 任务详情项目文件树顶层目录不全（大仓 max_files 截断）

## 现象

任务详情「项目文件树」展开仓库根（如 `ram-work`）后，只显示靠前的部分顶层目录/文件，后面的目录（如 `task2app`、`playwright` 等）缺失。

典型可见文本会在字母序中途截断，例如只到 `docs` / `ai.md` / `.gitignore` 附近。

## 环境与上下文

- 前端：`TaskDetailProjectFileTree.vue` → `GET …/cloud/compute/container-layer-files/?max_files=3000`
- 网关：`taskContainerGateway` L0 `container-layer-files` → 容器 `GET /api/layers/:id/files`
- 容器：`trae-agent/onlineServiceJS/src/layerFs.mjs` → `listFlatRelativeFilesForLayer`

## 根因

1. 列表实现为**深度优先**遍历，命中 `max_files`（默认/上限约 2000–5000）后立即停止。
2. 早期目录下的 `node_modules` 等噪声树可单独占满额度（如 `DaydaymoneyGrafana/node_modules/...`）。
3. 前端用扁平路径拼树：某顶层目录下若没有任何路径进入列表，该目录**完全不显示**。

## 修复

1. 跳过 `node_modules`、`.venv*`、`dist`/`build` 等列表噪声目录。
2. **先种子化全部顶层条目**（每顶层目录至少一个代表路径；空目录用 `dirname/` 标记），再 **BFS** 补齐。
3. 多仓并列时先对所有仓做顶层种子，避免前序仓深文件挤掉后序仓。
4. 响应增加 `truncated`；前端展示提示，并识别 `dirname/` 目录标记。

## 预防

- 大仓文件树不得依赖「一次 DFS 扁平列表必完整」；截断时必须保证**顶层可见**。
- 新增可占满磁盘的依赖目录时，同步加入 `SKIP_LISTING_DIR_NAMES`。
- 长期可演进为按需 `GET …/children` 懒加载（接口已存在），减少一次性扁平列表压力。
- 回归：`trae-agent/onlineServiceJS/src/layerFlatFiles.test.mjs`（顶层保留 / 多仓种子 / 跳过 node_modules）。

## 关联

- 组件：`taskFE/app/src/components/task-detail/TaskDetailProjectFileTree.vue`
- 列表实现：`trae-agent/onlineServiceJS/src/layerFs.mjs`
- 部署注意：容器镜像需包含上述 onlineServiceJS 改动；仅更新公网 SPA 不够。
