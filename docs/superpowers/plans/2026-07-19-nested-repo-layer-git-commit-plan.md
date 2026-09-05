# 实施计划：嵌套子仓层图提交

对应设计：`docs/superpowers/specs/2026-07-19-nested-repo-layer-git-commit-design.md`

## 任务

- [x] 1. 新增 `layerFsNestedGit.mjs`：内嵌 `.git` 有界扫描 + `expandGitWorkdirRootsWithNested`
- [x] 2. 改 `layerGitWorkdirRootsForFileListing` 调用 expand；补 `layerFs.nestedGitWorkdirs.test.mjs`
- [x] 3. `layerParentDiff.mjs`：最长 relPrefix 匹配
- [x] 4. 新增 `layerGitCommit.mjs` + 改 `server.mjs` commit 路由；单测
- [x] 5. `layerGitOauthPush.mjs`：仅 push dirty/ahead 根
- [x] 6. 跑相关 node:test；`layerFs` 拆分至 ≤500 行
- [x] 7. OPT 落盘与收尾

## NFR（L2）

- 扫描深度默认 8；跳过噪声目录，避免拖垮层图。
- push 过滤防止 30+ 子仓全量索 token。

## DDD / 事件

纯容器 git 操作，无新领域事件（书面例外：无 SaaS 业务状态变迁需 MQ）。
