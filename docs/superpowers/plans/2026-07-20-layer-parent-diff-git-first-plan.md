# 实施计划：OPT-20260720-018

## 任务

- [x] 抽出 `layerParentDiffCompare.mjs`（compareIndices 等），降低 `layerParentDiff.mjs` 行数
- [x] 新增 `layerParentDiffGit.mjs`：`collectPairChanges`（git-first + walk 回退）
- [x] `getLayerParentDiffFiles` 改用 `collectPairChanges`
- [x] 单测：共享 dirty、父脏子净、HEAD 分叉、非 git 回退 truncated
- [x] 更新意图文档阶段 C 为已落地
- [x] 标记 OPT-20260720-018 completed（原开放项误用 015，已避冲突重编号）

## 事件契约

无（容器只读，书面例外）。
