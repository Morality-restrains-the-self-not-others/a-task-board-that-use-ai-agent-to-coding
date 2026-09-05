# 设计文档：gitService GitLab 数据持久化（durable GITLAB_HOME）

- **日期**: 2026-07-22
- **作者**: claude（goal-mode 自动采用）
- **状态**: approved（goal-mode 跳过 USER GATE）
- **迭代**: gitservice-durable-gitlab-home

## 1. 问题陈述

gitService（GitLab CE 容器）在**容器重启或新建**后，用户感知为「数据丢失」：仓库、用户、OIDC/OAuth 配置状态看似回到空实例或半初始化状态。

## 2. 根因（已用运行时证据确认）

| 事实 | 证据 |
|------|------|
| compose 已 bind-mount `./gitlab_home/{config,logs,data}` | `gitService/docker-compose.yml` volumes |
| 当前数据目录落在工作区 | `docker inspect gitlab` → Source=`/tmp/ram-work/gitService/gitlab_home/...` |
| 工作区是 **tmpfs** | `findmnt -T /tmp/ram-work` → `tmpfs` |
| 容器内数据实际存在 | `docker exec gitlab du -sh /var/opt/gitlab` → ~592M |

**结论**：容器重启本身通常**不**擦除 bind-mount 内容；真正丢失发生在：

1. **宿主机重启 / ram-work 工作区重建** → tmpfs 清空 → `gitlab_home` 空目录被重新创建 → 新容器挂载空卷，表现为「新建后数据丢失」
2. **换 worktree / 新 clone 路径** → 相对路径 `./gitlab_home` 指向另一空目录
3. （次要）远程 Docker 时相对路径解析到 Docker 宿主机的不同目录

官方 GitLab Docker 文档约定使用宿主机持久目录 `$GITLAB_HOME`（如 `/srv/gitlab`），而非仓库内相对路径。

## 3. 成功标准（SMART）

1. 默认 `GITLAB_HOME` 位于**非 tmpfs** 文件系统（`$XDG_DATA_HOME/daydaymoney/gitService` 或 `$HOME/.local/share/daydaymoney/gitService`）
2. `docker compose` 卷路径使用 `${GITLAB_HOME}/…` 绝对持久目录
3. 首次切换时，若旧 `./gitlab_home` 有实质数据且新目录为空，**自动 rsync 迁移**后重建容器挂载
4. 若解析后的 `GITLAB_HOME` 仍在 tmpfs/ramfs，默认 **拒绝启动**（可用 `GITLAB_HOME_ALLOW_TMPFS=1` 覆盖）
5. `bash gitService/run.sh stop` / `stop --clean` **不删除** `GITLAB_HOME` 数据；仅 `--clean` 删容器与 bootstrap marks
6. 单测覆盖：路径解析、tmpfs 拒绝、迁移条件判定
7. 验收：停止容器 → `docker rm` → 再 `run.sh start` → 仓库列表与迁移前一致；且 `findmnt -T "$GITLAB_HOME"` 非 tmpfs

## 4. 方案选型（自动采用最优）

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 仅改文档提醒勿用 tmpfs | 零改动 | 无法防止默认踩坑 | ❌ |
| B. named Docker volume | 简单 | 备份/迁移/跨机不便；远程 Docker 行为不一 | ❌ |
| C. **durable `GITLAB_HOME` + 自动迁移 + tmpfs 门禁** | 对齐官方；跨重启/重建；可配置 | 需一次性迁移与脚本改动 | ✅ 采用 |

### 4.1 默认路径

```
GITLAB_HOME=${GITLAB_HOME:-${XDG_DATA_HOME:-$HOME/.local/share}/daydaymoney/gitService}
```

目录结构：

```
$GITLAB_HOME/
  config/
  logs/
  data/
  bootstrap_marks/
  .taskbill_admin_pat   # 既有 PAT 文件位置（随迁移）
```

配置覆盖（本服务 conf，非跨服务直读）：

```yaml
# conf/infra/git-service/config.yaml
gitlabHome: ""   # 空则用上述默认；可写绝对路径
```

### 4.2 启动流程变更（`run.sh`）

1. `resolve_gitlab_home` — env > conf `gitlabHome` > 默认
2. `assert_gitlab_home_durable` — tmpfs/ramfs 则失败（可覆盖）
3. `ensure_gitlab_home_dirs` + `migrate_legacy_gitlab_home_if_needed`
4. `export GITLAB_HOME` 后调用 `ensure_container` / compose
5. 兼容：仓库内 `gitService/gitlab_home` 若仍是目录且已迁移，保留；可选写 README 提示真实数据在 `GITLAB_HOME`

### 4.3 非目标

- 不引入外部对象存储备份
- 不改变 GitLab 镜像版本或 Omnibus 业务配置
- 不把 `stop --clean` 改为删除数据（需显式人工 `rm -rf "$GITLAB_HOME"`）

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 确保 GitLab 持久卷可用 | — | — | — | 基础设施运维；无跨聚合业务状态变更，**无对应 MQ 事件** |
| 从 legacy 目录迁移数据 | GitLabHomeMigrated（可选本地日志事件） | `run.sh` 日志 | 运维观测 | 不投递业务 MQ；仅结构化日志 |

## 6. Domain Concept Inventory

- **Bounded Context**: gitService / GitLab hosting（infra）
- **Entity**: GitLabInstance（已有 `gitService/domain/`）
- **Value Object**: `GitLabHomePath`（绝对路径、fstype、durable 标志）
- **Domain Event（运维）**: `GitLabHomeMigrated`（日志级，非业务 MQ）

## 7. 价值流影响

- 不新增业务价值流步骤；加固所有依赖系统内建 GitLab 的流（创建项目、OAuth bind、磁盘配额同步）的数据耐久性。
- 字段：无 DB schema 变更。

## 8. 🏛️ 架构变更影响

- **迭代版本**: v53 🎯 target
- **迭代名称**: gitservice-durable-gitlab-home
- **变更明细**:
  - 🟡 [MODIFIED] gitService / GitLab CE — 数据卷从仓库相对 `./gitlab_home`（常位于 tmpfs worktree）改为宿主机 durable `GITLAB_HOME`
  - 🟢 [NEW] Technology Artifact `GitLabHomeDurableStore`
  - ✅ [CLOSES] Gap: GitLab 数据随 tmpfs/worktree 销毁而丢失
- **新增文件**:
  - `docs/architecture/v53-application-integration-20260722-2235-claude.{puml,archimate,mermaid.md}`

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| Plateau v52 | current — 应用集成基线；GitLab 卷仍相对路径 |
| Plateau v53 | target — durable GITLAB_HOME |
| Gap | tmpfs/worktree 绑定导致重建丢数 |
| WorkPackage | 解析路径 + 迁移 + tmpfs 门禁 + compose 变量化 |

## 9. 风险与回滚

| 风险 | 缓解 |
|------|------|
| 迁移中途断电 | rsync 后校验 `data/postgresql` 存在再 `docker rm`+`up`；失败保留 legacy |
| 磁盘空间不足 | 迁移前 `df` 检查需 ≥ 源目录 1.2× |
| 用户自定义路径在 NFS | 允许；仅拒绝 tmpfs/ramfs |
| 旧脚本硬编码 `gitlab_home` | PAT 脚本改为 `${GITLAB_HOME:-…}`；迁移后 symlink 可选 |

## 10. 测试计划

1. 单元：`scripts/test_gitlab_home_resolve.sh`（或 pytest/bash）覆盖 resolve / tmpfs / migrate predicate
2. 手工/集成：记录项目 ID → stop → rm 容器 → start → 项目仍在；`findmnt` 非 tmpfs
3. 回归：OIDC bootstrap marks 在新 `GITLAB_HOME/bootstrap_marks` 或沿用 `.bootstrap_marks` 策略保持幂等
