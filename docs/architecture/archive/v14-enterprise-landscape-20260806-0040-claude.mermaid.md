# v14 Enterprise Landscape — 多智能体会话互知与冲突防护 (Target, 2026-08-06 00:40)

```mermaid
graph TD;
  developer["Developer"];
  hooksSSOT["Git Hooks 模板 SSOT"];
  hooksDeploy["Hooks 分发/校验器 [MODIFIED v14]"];
  repoGitHooks[".githooks/ 入库真源 ×38 仓"];
  oldHookCopy[".git/hooks/ 复制模式 [DEPRECATED v13]"];
  gitlab["GitLab CE (:8012)"];
  sessionHub["Session Hub (logs/sessions/)"];
  sessionCLI["claude-agent session CLI (Go)"];
  ccHooks["Claude Code 会话钩子"];
  shadowWS["Shadow Edit 工作区"];
  submodulePool["开发子仓池 (40+ 子模块)"];
  plateauV13["Plateau v13 — Git Hooks 入库"];
  plateauV14["Plateau v14 — 多会话互知"];
  gapSession["Gap: 多会话并发无互知无锁"];
  wpSession["WP-session-coordination"];
  hooksSSOT --> hooksDeploy;
  hooksDeploy --> repoGitHooks;
  developer --> hooksDeploy;
  repoGitHooks --> gitlab;
  oldHookCopy --- repoGitHooks;
  hooksDeploy --> ccHooks;
  sessionCLI --> sessionHub;
  ccHooks --> sessionCLI;
  ccHooks --> sessionHub;
  sessionCLI --> shadowWS;
  sessionHub --> submodulePool;
  developer --- ccHooks;
  plateauV13 --> gapSession;
  wpSession --|> gapSession;
  wpSession --|> plateauV14;
```
