# 实施计划: git-service 容器复用快速重启

> 输入:
> - 设计文档: `.claude/plans/git-service-fast-restart-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-git-service-fast-restart-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-30-git-service-fast-restart-nfr-clarification.md`
>
> 总代码量: ~1 文件修改（`gitService/run.sh`），~80 行新增/修改

---

## 任务清单

### Increment 1: 容器复用 — ensure_container 逻辑

- [ ] **1.1** 在 `gitService/run.sh` 中新增 `wait_for_gitlab_ready()` 函数
  - 路径: `gitService/run.sh`（在 `cleanup_previous_stack()` 之后）
  - 参数: `$1` = max_wait (默认 300)
  - 轮询: `docker exec gitlab gitlab-rails runner "puts 'ready'"`，每 5s 一次
  - 超时返回 1，成功返回 0
  - **验证**: `bash -c 'source gitService/run.sh; wait_for_gitlab_ready 10'`（容器运行时应秒级返回）

- [ ] **1.2** 在 `gitService/run.sh` 中新增 `ensure_container()` 函数
  - 路径: `gitService/run.sh`
  - 三种分支:
    - 容器已运行 (`docker ps` 匹配) → echo "复用已有容器"，return 0
    - 容器已停止 (`docker ps -a` 匹配) → `compose_file start gitlab` → `wait_for_gitlab_ready` → return 0
    - 容器不存在 → `compose_file up -d` → sleep 3 → `NEED_BOOTSTRAP=true`
  - 环境变量: `GITLAB_CONTAINER="${GITLAB_CONTAINER:-gitlab}"`
  - **验证**: 三种容器状态下 `ensure_container` 均正常返回

- [ ] **1.3** 修改 `run.sh` 主流程中的 `start` 和 `managed` case
  - 路径: `gitService/run.sh`，约第 246-263 行
  - 替换 `cleanup_previous_stack` + `compose_file up -d` 为 `ensure_container`
  - **不删除** `cleanup_previous_stack()` 函数定义（留作 fallback）
  - **验证**: `bash gitService/run.sh start` 在已有容器时跳过重建

- [ ] **1.4** 修改 `stop` case 为 `docker compose stop`（保留容器）
  - 路径: `gitService/run.sh`，约第 241-244 行
  - 默认: `compose_file stop`
  - `--clean` 标志: `compose_file down --remove-orphans`
  - 打印提示: "容器已停止。使用 stop --clean 可完全清理。"
  - **验证**: `bash gitService/run.sh stop` 后容器消失但 `docker ps -a` 仍可见

### Increment 2: Bootstrap 标记跳过

- [ ] **2.1** 在 `docker-compose.yml` 中新增 bootstrap marks 卷挂载
  - 路径: `gitService/docker-compose.yml`
  - 新增: `- './gitlab_home/bootstrap_marks:/opt/gitlab/bootstrap_marks'`
  - **验证**: `docker compose config` 无语法错误

- [ ] **2.2** 在 `run.sh` 中新增 `run_bootstrap_if_needed()` 函数
  - 路径: `gitService/run.sh`
  - 标记目录: `BOOTSTRAP_MARKS_DIR="./gitlab_home/bootstrap_marks"`
  - 三个脚本均条件执行:
    - `sync_local_oauth_app_scopes.sh` — 标记 `oauth_scopes_synced`
    - `sync_omniauth_oidc.sh` — 标记 `omniauth_oidc_synced`
    - `fix_oidc_ssl.sh` — 标记 `oidc_ssl_fixed`
  - 首次创建 (`NEED_BOOTSTRAP=true`) → 全部执行 + touch 标记
  - 标记存在 → echo "已同步，跳过" + return
  - **验证**: 二次启动时三个脚本均打印 "跳过"

- [ ] **2.3** 修改主流程，将 bootstrap 调用替换为 `run_bootstrap_if_needed`
  - 路径: `gitService/run.sh`，约第 284-295 行
  - 替换三个硬编码的脚本调用为 `run_bootstrap_if_needed`
  - **验证**: `bash gitService/run.sh start` 首次执行 bootstrap，再次执行全部跳过

### Increment 3: stop 语义 + Fallback

- [ ] **3.1** 新增 fallback 逻辑：`compose_file start` 失败时自动重建
  - 路径: `gitService/run.sh`，`ensure_container()` 函数内
  - `compose_file start gitlab` 返回非零 → echo "容器无法启动，重建中..." → `cleanup_previous_stack` → `compose_file up -d` → `NEED_BOOTSTRAP=true`
  - **验证**: 模拟损坏容器（`docker cp /dev/null gitlab:/tmp` 后 `docker stop gitlab`），重启应自动重建

- [ ] **3.2** 确认 runAll 集成：`managed` 模式端到端
  - 路径: 无需修改 `conf/runAll.yaml`
  - **验证**: `cd runAll && ./bin/runAll --config ../conf/runAll.yaml --services git-service`
    - 首次: 完整初始化 (~4.5 min)，health check 通过
    - 二次: 秒级复用 (<5s)，health check 通过

- [ ] **3.3** 手动测试：三种场景验收
  - 场景 A: 全新部署（无容器）→ 正常创建 + bootstrap
  - 场景 B: 重启（stop → start）→ 秒级复用，跳过 bootstrap
  - 场景 C: 容器损坏恢复 → 自动重建 + bootstrap

---

## 文件变更清单

| 文件 | 操作 | 行数估计 |
|------|------|---------|
| `gitService/run.sh` | 修改 | +70 / -10 |
| `gitService/docker-compose.yml` | 微调 | +1 |

## 风险与回滚

- **回滚**: `git checkout gitService/run.sh gitService/docker-compose.yml`
- **数据安全**: `gitlab_home/` 持久化卷不变，仓库/DB 数据零影响
- **向后兼容**: `run.sh start` / `stop` / `managed` 调用签名不变，仅内部优化
