# Value Stream: git-service 容器复用快速重启

> Derived from design: `.claude/plans/git-service-fast-restart-design.md`

## Value Summary

开发者在 runAll UI 重启 git-service 时，已有容器直接复用（秒级就绪），不再重建容器和重新执行 bootstrap 脚本，从而缩短 git-oauth / saas-backend 等下游依赖的等待时间。

## Related Value Streams

- **docker-infra-no-repull**：同属 Docker 基础设施启动优化；该变更解决镜像重复拉取，本变更解决容器重复创建。
- **runall-cascade-lifecycle**：链式启动依赖 git-service 先 healthy；本变更缩短其上游等待（~4.5 min → <5s）。
- **runall-docker-infra-split**：基础设施分组管理；git-service 归属 infrastructure group。
- Greenfield 增量 — 无既有 git-service 容器生命周期专用 value stream。

## End-to-End Flow

```
[开发者点启动/重启 git-service]
  → [runAll 触发 run.sh start]
  → [检查容器状态: 已运行? 已停止? 不存在?]
  → [已运行: 直接复用 / 已停止: compose start / 不存在: compose up -d 创建]
  → [Bootstrap 脚本: 仅在首次创建或配置变更时执行]
  → [健康检查: /users/sign_in 可达]
  → [git-oauth / saas-backend 可链式启动]
```

## Value Increments

### Increment 1: 容器复用 — ensure_container 逻辑（Thin Slice）
**Value to user（开发者）:** 重启 git-service 时，已有容器 <5 秒复用，不再等待 4.5 分钟
**Scope:** `gitService/run.sh` — 新增 `ensure_container()` + `wait_for_gitlab_ready()` 函数；stop 改为 `compose stop`
**Depends on:** nothing

### Increment 2: Bootstrap 标记跳过
**Value to user（开发者）:** 重启时跳过已执行的三步 bootstrap 脚本（OAuth scope sync / OIDC reconfigure / SSL fix）
**Scope:** `gitService/run.sh` — 新增 `run_bootstrap_if_needed()` + 标记文件检查
**Depends on:** Increment 1

### Increment 3: stop 语义调整（可选）
**Value to user（开发者）:** `stop` 保留容器不删除，`stop --clean` 可达旧行为
**Scope:** `gitService/run.sh` — stop case 分支调整
**Depends on:** Increment 1
