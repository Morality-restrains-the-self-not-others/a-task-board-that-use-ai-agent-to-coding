# 价值流：父层 diff git-first

```
用户打开任务详情「文件变动」
  → Gateway 调 container-layer-diff-parent-files
  → onlineService getLayerParentDiffFiles
  → [新] git 候选路径 → 分类
  → [回退] 无 git 时 collectIndex
  → 分页 + enrichChangesWithGitStatus
  → 前端列表 / 琥珀截断提示（仅 walk 触顶时）
```

最小可验证增量：单测证明「大量无关文件 + 少量 git dirty」不 truncated，且列出 dirty 路径。
