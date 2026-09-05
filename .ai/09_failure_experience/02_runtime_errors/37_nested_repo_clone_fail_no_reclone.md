# [运行时] 子仓库克隆失败行无具体错误与重新克隆

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

任务详情「子仓库克隆状态」失败行仅见徽章「克隆失败」+ path/短 URL（如 `克隆失败relayToTraeljy…/relayToTrae`），无具体 git 错误文案，亦无「重新克隆」按钮。

复现：`data-testid="task-nested-repos-clone-row"` 失败态。

## 根因

1. `TaskDetailNestedReposCloneStatus` 已用进度 message 判定 `kind===error`，但未渲染 `status.message`。
2. 未接线父级 `repo-reclone`；且 Django `forward_container_repo_reclone` allowlist 仅含 `project.git_repos`，子仓 URL 会被 400 拒绝。

## 解决方案

- 前端：失败行展示 `formatNestedRepoCloneErrorDetail`；`shouldShowNestedRepoRecloneButton` + 按钮 emit `{ repoUrl, parentRepoUrl, cloneAlias }`。
- `onRepoReclone`：支持 payload；身份回退父仓；请求体带 `parent_repo_url` / `clone_alias`。
- 后端：父仓属工作空间时，经 `list_nested_git_repos` 校验子仓后放行；转发 `clone_alias` 到容器 `/api/repos/reclone`。

## 预防

- 失败态 UI 必须同时给出原因与可行动作（重试），禁止仅徽章。
- 子仓不入 `project_repos` 时，所有「按仓库 URL 鉴权」的接口须显式支持 nested 归属校验。

## 验证

```bash
cd taskFE/app && npx vitest run \
  src/utils/nestedRepoCloneStatusUtils.test.js \
  src/components/task-detail/TaskDetailNestedReposCloneStatus.test.js
cd task2app/Saas_project && ../activate_env.sh run -- \
  python -m pytest cloud/tests/test_forward_container_repo_reclone_nested.py -q
```

公网：`bash scripts/runall-lifecycle.sh build` 后硬刷新任务详情；确认失败行有错误文案与「重新克隆」。
