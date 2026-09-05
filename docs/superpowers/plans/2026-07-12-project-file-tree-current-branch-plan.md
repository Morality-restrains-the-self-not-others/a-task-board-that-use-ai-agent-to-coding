# 实施计划：文件树提交日志旁显示当前分支

- 日期：2026-07-12

## 任务

- [x] 1. Intent 018 + 索引
- [x] 2. `layerFs.mjs`：`clickedPathIsGitRepoRoot` + 单测
- [x] 3. `server.mjs`：`git/log` 附带 `is_repo_root` / `current_branch`
- [x] 4. Gateway：`container-layer-git-log` 转发 `path`/`limit` + 测试
- [x] 5. Vue：`TaskDetailProjectFileTree` + `TaskDetailExecLayerChangePreview`
- [x] 6. Playwright：mock 返回分支并断言 UI；子目录不显示
- [x] 7. 跑 onlineServiceJS 相关单测 + 前端相关测

## 验证命令

```bash
cd trae-agent/onlineServiceJS && node --test src/layerFlatFiles.test.mjs
cd taskContainerGateway && go test ./src -run GitLog -count=1
# Playwright（mock）：TaskDetail.project-file-tree.playwright.test.js
```
