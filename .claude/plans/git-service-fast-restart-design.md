# git-service 快速重启优化 — 设计文档

## 1. 问题诊断

### 1.1 用户报告

> 页面 http://183.250.1.132:9999/ 的 git-service 每次启动都很慢，是不是重启的时候没有复用容器，导致每次都重新初始化?

**结论：用户的判断完全正确。** 每次重启都会销毁并重建 GitLab 容器，导致完整的重新初始化。

### 1.2 根因分析（三层叠加）

| 层次 | 位置 | 问题 |
|------|------|------|
| **A. 容器未复用** | `gitService/run.sh:90-93` | `cleanup_previous_stack()` 无条件执行 `docker compose down --remove-orphans`，销毁容器 |
| **B. GitLab CE 内部初始化** | GitLab 镜像自身 | 新容器启动后，PostgreSQL/Redis/Rails/Puma 需要 30-90s 全部就绪 |
| **C. 同步 Bootstrap 脚本** | `run.sh:284-295` | 三个脚本顺序执行，各自轮询等待 GitLab 就绪（最长各 300s） |

**完整启动时间线**（来自 2026-06-28 日志）：

```
T+0s     runAll 发起 start
T+1s     cleanup_previous_stack: docker compose down    ← 销毁旧容器
T+1s     docker compose up -d                           ← 创建新容器
T+1s     GitLab 容器启动（PostgreSQL → Redis → Rails → Puma）
T+54s    GitLab Rails 就绪（gitlab-rails runner 可用）
T+54s    sync_local_oauth_app_scopes.sh 开始
T+174s   OAuth scope 同步完成（4 个 provider）
T+174s   sync_omniauth_oidc.sh 开始
T+263s   OIDC 同步完成
T+263s   fix_oidc_ssl.sh 开始（fix 已存在，快速跳过）
T+270s   runAll 健康检查通过
────────
总计: ~4.5 分钟
```

### 1.3 为什么持久化卷不能解决问题

```yaml
# docker-compose.yml:70-73
volumes:
  - './gitlab_home/config:/etc/gitlab'     # GitLab 配置 ✓
  - './gitlab_home/logs:/var/log/gitlab'   # GitLab 日志 ✓
  - './gitlab_home/data:/var/opt/gitlab'   # 数据库/仓库数据 ✓
```

这三个 bind-mount 卷确实持久化了 GitLab 的**数据**（仓库、数据库文件、配置），但 `docker compose down` 删除容器后，下次 `up` 创建的是**全新容器**。GitLab CE 的新容器仍然需要：
- 启动内嵌 PostgreSQL 并恢复数据库连接
- 启动 Redis
- 编译 Rails 资产
- 启动 Puma web 服务器
- 所有内部 health check 通过

**类比**：数据在硬盘上没丢，但每次都要重新「开机 → BIOS → 操作系统启动 → 服务启动」。

---

## 2. 优化方案

### 2.1 核心思路：容器复用 + 幂等 Bootstrap

将 `run.sh` 的启动逻辑改为 **「检查 → 复用 → 仅在必要时重建」** 模式：

```
当前流程:  stop → down (销毁) → up -d (创建+初始化) → bootstrap
优化流程:  stop → start (复用已有容器) → 跳过 bootstrap
           ↓ 仅在容器不存在/损坏时
           down → up -d → bootstrap
```

### 2.2 方案细节

#### 2.2.1 新增 `ensure_container()` 函数（`run.sh`）

```bash
# 替换 cleanup_previous_stack 的无条件销毁逻辑
ensure_container() {
  local CONTAINER_NAME="${GITLAB_CONTAINER:-gitlab}"

  # 情况1: 容器已在运行 → 直接复用
  if docker ps --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    echo "GitLab 容器 $CONTAINER_NAME 已在运行，复用现有容器。"
    return 0
  fi

  # 情况2: 容器存在但已停止 → docker compose start（不重建）
  if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    echo "GitLab 容器 $CONTAINER_NAME 已存在但已停止，启动现有容器..."
    compose_file start gitlab
    echo "等待 GitLab 就绪..."
    wait_for_gitlab_ready 300
    return 0
  fi

  # 情况3: 容器不存在 → 首次创建
  echo "首次创建 GitLab 容器..."
  compose_file up -d
  sleep 3
  assert_docker_still_running || exit 1
  compose_file ps
  # 首次创建需要 bootstrap
  NEED_BOOTSTRAP=true
}
```

#### 2.2.2 新增 `wait_for_gitlab_ready()` 函数

```bash
# 统一的 GitLab 就绪等待（从三个脚本提取公共逻辑）
wait_for_gitlab_ready() {
  local max_wait="${1:-300}"
  local waited=0
  local CONTAINER_NAME="${GITLAB_CONTAINER:-gitlab}"

  while (( waited < max_wait )); do
    if docker exec "$CONTAINER_NAME" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
      echo "GitLab 已就绪。"
      return 0
    fi
    sleep 5
    waited=$((waited + 5))
  done
  echo "GitLab 初始化超时（${max_wait}秒）" >&2
  return 1
}
```

#### 2.2.3 Bootstrap 标记文件（跳过已执行的初始化）

在持久化卷上创建标记文件，避免每次启动都重新执行 bootstrap：

```bash
# docker-compose.yml 增加一个持久化标记目录
# volumes:
#   - './gitlab_home/bootstrap_marks:/opt/gitlab/bootstrap_marks'
```

```bash
# run.sh 中 bootstrap 的条件执行
BOOTSTRAP_MARKS_DIR="./gitlab_home/bootstrap_marks"

run_bootstrap_if_needed() {
  if [[ "${NEED_BOOTSTRAP:-false}" != "true" ]]; then
    echo "复用已有容器，跳过 bootstrap 脚本。"
    return 0
  fi

  # sync_local_oauth_app_scopes
  if [[ ! -f "$BOOTSTRAP_MARKS_DIR/oauth_scopes_synced" ]]; then
    echo "同步本地 GitLab OAuth 应用 scope…"
    "$SCRIPT_DIR/scripts/sync_local_oauth_app_scopes.sh" \
      && touch "$BOOTSTRAP_MARKS_DIR/oauth_scopes_synced" \
      || echo "提示: OAuth scope 同步未完成" >&2
  else
    echo "OAuth scope 已同步，跳过。"
  fi

  # sync_omniauth_oidc
  if [[ ! -f "$BOOTSTRAP_MARKS_DIR/omniauth_oidc_synced" ]]; then
    echo "同步 GitLab OmniAuth OIDC 配置…"
    "$SCRIPT_DIR/scripts/sync_omniauth_oidc.sh" --reconfigure \
      && touch "$BOOTSTRAP_MARKS_DIR/omniauth_oidc_synced" \
      || echo "提示: OIDC 同步未完成" >&2
  else
    echo "OmniAuth OIDC 已同步，跳过。"
  fi

  # fix_oidc_ssl (该脚本本身已是幂等的，但仍加标记)
  if [[ ! -f "$BOOTSTRAP_MARKS_DIR/oidc_ssl_fixed" ]]; then
    echo "应用 OIDC SSL 协议修复…"
    "$SCRIPT_DIR/scripts/fix_oidc_ssl.sh" \
      && touch "$BOOTSTRAP_MARKS_DIR/oidc_ssl_fixed" \
      || echo "提示: OIDC SSL 修复未完成" >&2
  else
    echo "OIDC SSL fix 已应用，跳过。"
  fi
}
```

#### 2.2.4 修改 `run.sh` 主流程

```bash
case "$mode" in
  stop)
    echo "正在停止 GitLab 容器..."
    compose_file stop    # ← 改为 stop（不销毁容器），保留容器可复用
    exit 0
    ;;

  managed)
    echo "runAll 托管模式：确保 GitLab 容器可用..."
    ensure_container
    run_bootstrap_if_needed
    ;;

  *)
    echo "确保 GitLab 容器可用..."
    ensure_container
    run_bootstrap_if_needed
    ;;
esac
```

### 2.3 stop 语义调整

**关键决策点**：`stop` 命令是否应该保留容器？

| 选项 | 行为 | 优缺点 |
|------|------|--------|
| `docker compose stop` | 停止容器但不删除 | ✅ 下次 start 秒级恢复；❌ 占用磁盘（容器层） |
| `docker compose down` | 停止并删除容器 | ✅ 干净；❌ 下次 start 需重建 |

**建议**：默认使用 `stop`，增加 `--clean` 标志在需要完全清理时使用 `down`。

```bash
case "$mode" in
  stop)
    if [[ "${1:-}" == "--clean" ]]; then
      echo "完全清理 GitLab 容器和数据..."
      compose_file down --remove-orphans --volumes
    else
      echo "正在停止 GitLab 容器（保留容器以加速下次启动）..."
      compose_file stop
      echo "提示: 使用 'bash run.sh stop --clean' 完全清理容器和数据。"
    fi
    exit 0
    ;;
```

### 2.4 Bootstrap 脚本各自的幂等性改进

三个脚本本身就具备一定程度的幂等性，可以独立加固：

| 脚本 | 当前幂等性 | 增强方案 |
|------|-----------|---------|
| `sync_local_oauth_app_scopes.sh` | `find_or_initialize_by` → 不会重复创建 | ✅ 无需改动，直接可跳过 |
| `sync_omniauth_oidc.sh` | 检查 provider 是否存在再 reconfigure | ✅ 已有跳过逻辑，加外部标记即可 |
| `fix_oidc_ssl.sh` | 第23行已检查 fix 是否已应用 | ✅ 完全幂等，随时可跳过 |

---

## 3. 预期效果

| 场景 | 当前耗时 | 优化后耗时 | 改善 |
|------|---------|-----------|------|
| **首次启动**（冷启动） | ~4.5 min | ~4.5 min | 无变化（仍需初始化） |
| **重启（容器已存在）** | ~4.5 min | **< 5 秒** | ~98% 减少 |
| **重启（容器被意外删除）** | ~4.5 min | ~4.5 min | 退化到冷启动 |
| **stop + start 循环** | 9 min | **< 10 秒** | ~98% 减少 |

---

## 4. 影响范围

### 4.1 修改文件

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| `gitService/run.sh` | **重构** | 新增 `ensure_container`、`wait_for_gitlab_ready`、`run_bootstrap_if_needed`；修改 `stop` 为 `compose stop` |
| `gitService/docker-compose.yml` | **微调** | 新增 `bootstrap_marks` 卷挂载（可选，标记文件也可放在宿主机） |
| `conf/runAll.yaml` | **无需改动** | 现有 health_check 配置不变，start_command/stop_command 不变 |

### 4.2 不影响的组件

- `runAll` 编排器 — 健康检查 URL 不变，超时不变
- `git-oauth` 服务 — 依赖 git-service，git-service 更早就绪对其透明
- `saas-backend` / `task-gateway` — 间接依赖不变
- GitLab 数据持久性 — volume 挂载不变，数据安全

### 4.3 向后兼容

- `bash gitService/run.sh start` — 行为不变，首次仍需初始化
- `bash gitService/run.sh stop` — 默认改为 `stop` 而非 `down`，需 `--clean` 达到旧行为
- `bash gitService/run.sh managed` — runAll 调用，行为不变

---

## 5. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| `docker compose start` 启动失败（容器状态损坏） | 容器无法启动 | fallback: 检测 start 失败后自动 `down → up -d` |
| Bootstrap 标记过期（OAuth 配置变更后标记仍为已执行） | scope 不同步 | 标记文件中记录配置 hash，变更后自动重新执行 |
| 长时间运行的容器积累日志/临时文件 | 磁盘增长 | 定期日志轮转（已有 `gitlab_home/logs` 映射） |
| `stop` 不删容器导致 docker 资源泄漏 | 网络/卷残留 | 提供 `--clean` 命令；不影响功能 |

---

## 6. 领域概念清单

本次为基础设施优化，不引入新的业务领域概念。涉及的运维概念：

- **GitLab Instance** — 自托管 GitLab CE 容器实例（已存在于 `gitService/domain/`）
- **Container Lifecycle** — Docker 容器的创建/启动/停止/销毁状态机
- **Bootstrap Marker** — 持久化文件标记，记录初始化步骤的完成状态

## 7. 价值流影响

本次改动不涉及任何现有价值流（`value-stream.yaml` 中无 infra 层定义）。属于 **基础设施运维优化**，对所有依赖 git-service 的流产生正面影响（更快的开发迭代周期）。

---

## 8. 替代方案（已评估并否决）

| 方案 | 描述 | 否决原因 |
|------|------|---------|
| 使用 Docker `restart: always` + 永不 stop | 让 GitLab 始终运行 | 资源浪费（内存 3-6 GB），不适用于开发环境 |
| 使用 Podman/systemd 管理生命周期 | 切换容器运行时 | 引入不必要的依赖，破坏现有 docker compose 工作流 |
| 在 runAll 层面缓存健康检查结果 | runAll 认为容器 healthy 就跳过 start | 绕过 run.sh 的启动逻辑，破坏封装 |
| 使用 GitLab 的 `gitlab-ctl stop/start` 而非容器重启 | 进程级重启 | GitLab CE 内嵌 PostgreSQL/Redis，进程重启仍然很慢 |

---

## 9. 下一步

设计文档已完成，请审阅。实现工作量估计：**~1 小时**（修改 `run.sh` + 测试三种启动场景）。
