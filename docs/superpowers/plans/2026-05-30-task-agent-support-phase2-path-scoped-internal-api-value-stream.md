# Value Stream: taskAgentSupport Phase 2 路径化 Internal API

> Derived from design: `docs/superpowers/specs/2026-05-30-task-agent-support-phase2-path-scoped-internal-api-design.md`

## Value Summary

容器 relay 直启时，onlineServiceJS 经 taskAgentSupport 网关完成 bootstrap（task-detail → repo-clone-credentials → feature-params-env → clone），运维可在 Internal API 404/日志中直接看到 tenant/workspace/task scope。

## Related Value Streams

- **task-detail-runtime-relay**（modification）：bootstrap 404 阻断 relay 直启，本流修复 internal 转发链
- **task-detail-repo-clone-credentials-decoupling**（extension）：repo-clone-credentials internal 转发可用
- **relay-token-audit-observability**（extension）：internal 访问 path 含 scope，排障更清晰

## End-to-End Flow

[用户点击 relay 直启] → [go_relay 换票 exchange-refresh ✓] → [onlineServiceJS bootstrap task-detail] → [repo-clone-credentials] → [feature-params-env] → [git clone + git-clone-progress] → [用户看到容器在线 + 仓库克隆完成]

## Value Increments

### Increment 1: 路径化 Internal + task-detail（Thin Slice）
**Value to user:** relay 直启不再因 task-detail 404 失败  
**Scope:** Django scoped url + dispatch 重构；Go forward URL；注册 task-detail  
**Depends on:** Phase 1 taskAgentSupport 网关

### Increment 2: Bootstrap 全量（Phase 2 actions）
**Value to user:** bootstrap 完整走完 clone 前置链  
**Scope:** repo-clone-credentials、feature-params-env、git-clone-progress  
**Depends on:** Increment 1

### Increment 3: 运行时同步（Phase 3 actions）
**Value to user:** zTree / layer OAuth / diff 上报经网关可用  
**Scope:** layer-graph-push、layer-changes-push、layer-github-oauth-access-tokens  
**Depends on:** Increment 2
