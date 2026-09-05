# v13 Enterprise Landscape — Mermaid Diagram

> 架构版本: v13 ✅ current | 作者: claude | 日期: 2026-08-06 00:08
> 迭代: git-hooks-version-control
> 基于: v12 (2026-08-05) | Git Hooks 入库与统一管理

```mermaid
graph TD;
  subgraph Business
    developer["Developer"]
  end
  subgraph Technology
    hooksSSOT["Git Hooks 模板 SSOT (scripts/hooks/templates/)"]
    hooksDeploy["Hooks 分发/校验器 (install-hooks-all.sh + --check/deploy, HOOK_VERSION)"]
    repoGitHooks[".githooks/ 入库真源 ×38 仓 (core.hooksPath)"]
    gitlab["GitLab CE (:8012)"]
  end
  subgraph Implementation
    plateauV12["Plateau v12 — 基线回填 (2026-08-05)"]
    plateauV13["✅ Plateau v13 — Git Hooks 入库 (2026-08-06)"]
    gapHooks["Gap: 钩子不入库 → 内容/部署/版本不可跟踪"]
    wpHooks["WP-git-hooks-vc (P0→P1→P2→P3→P4)"]
  end

  developer -- "一键激活 install-hooks-all.sh" --> hooksDeploy
  hooksSSOT -- "模板渲染 + 版本戳" --> hooksDeploy
  hooksDeploy -- "分发/校验 (38 仓 .githooks/ + hooksPath)" --> repoGitHooks
  repoGitHooks -- "钩子入库提交" --> gitlab

  plateauV12 -- "identifies" --> gapHooks
  wpHooks --|> "closes" gapHooks
  wpHooks --|> "delivers" plateauV13
```
