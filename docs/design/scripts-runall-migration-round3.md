# scripts/ → runAll 归属分析（第三轮）

> 头脑风暴产物 — 设计文档
> 日期: 2026-06-05
> 状态: 待审批

---

## 当前 scripts/ 状态（12 个文件）

```
scripts/
├── ci/
│   ├── check_conf_sync.sh           ← 全域 CI，不过属于 conf/
│   └── check_ddd_bdd_compliance.py  ← 全域 CI
├── docker-env.sh                    ← 基础设施层
├── docker-desktop-helper.sh         ← Mac 辅助
├── docker-shell.sh                  ← 终端集成 (⚠️ 有死代码)
├── docker-setup.sh                  ← 开发者初始化
├── docker-status.sh                 ← 状态查询
├── docker-tunnel-remote.sh          ← SSH 隧道
├── docker-context-init.sh           ← Context 初始化
├── docker-install-cli.sh            ← CLI 安装
├── docker-uninstall-desktop.sh      ← Desktop 卸载
├── docker-use-remote.sh             ← 切换远程 context
└── runall-local-promtail.sh         ← runAll 日志采集
```

---

## 逐脚本分析

### 明确属于 runAll：runall-local-promtail.sh

| 维度 | 结论 |
|------|------|
| 命名 | 文件名以 `runall` 开头 |
| 职能 | 管理 runAll 日志管道（tail 本地日志 → push 远程 Loki） |
| runAll 配置引用 | `conf/runAll.yaml:15` 注释指出 "bash scripts/runall-local-promtail.sh up" |
| 依赖 | 调用 `docker-env.sh` 获取 `DOCKER_CTX_LOCAL`；引用 `AiMonitor/docker-compose.promtail-local.yaml` |

**判断**: ✅ 应移至 `runAll/scripts/runall-local-promtail.sh`

**迁移影响**:
- ROOT 计算: `dirname $0/..` → `dirname $0/../..`（从 runAll/scripts/ 回退到 monorepo root）
- docker-env.sh 引用: `$ROOT/scripts/docker-env.sh` → `$ROOT/runAll/scripts/docker-env.sh`（如果 docker-env 也移动）或保持（如果 docker-env 不动）
- runAll.yaml 注释: 更新路径

### Debatable：docker-* 脚本组（9 个文件）

这 9 个脚本形成紧密耦合的 Docker 环境管理层：

```
docker-env.sh              ← 被 6 个脚本 source
├── docker-use-remote.sh   ← source docker-env.sh
├── docker-status.sh       ← source docker-env.sh
├── docker-tunnel-remote.sh ← source docker-env.sh
├── docker-context-init.sh ← source docker-env.sh
├── docker-uninstall-desktop.sh ← source docker-env.sh
└── runall-local-promtail.sh ← source docker-env.sh (将迁移)

docker-shell.sh            ← 被 ~/.zshrc source
docker-setup.sh            ← 手动执行
docker-install-cli.sh      ← 手动执行
docker-desktop-helper.sh   ← 被 docker-setup 调用
```

**属于 runAll 的理由**:
- Docker 环境是 runAll 编排能力的基础设施
- runAll 的远程 Docker 工作流依赖这些脚本
- 遵循「服务基础设施归服务所有」原则

**不属于 runAll 的理由**:
- 它们是**开发者本机环境**工具（配置 `~/.zshrc`、安装 CLI、管理 SSH）
- 不参与任何服务的编译或运行时——纯开发环境 setup
- 开发者可能单独使用 Docker 而不启动 runAll
- 它们形成一个内聚组——拆分会破坏这种内聚性

**判断**: 

两种方案都合理。推荐：

| 方案 | 做法 | 优点 | 缺点 |
|------|------|------|------|
| A（推荐） | 只移动 runall-local-promtail.sh | 最小变更，docker 脚本保持内聚 | scripts/ 还有 9 个 docker 脚本 |
| B | 全量迁移 docker-* 到 runAll/scripts/ | scripts/ 只剩 CI，极度精简 | 需更新 ~10 处内部交叉引用 + ~/.zshrc 示例 |

### ⚠️ 附带发现：docker-shell.sh 中的死代码

`docker-shell.sh` 的以下函数/别名指向已删除的 `runall-remote-docker.sh`：

```bash
docker-infra-remote() {
  "$_ram_mount_docker_scripts/runall-remote-docker.sh" infra "$@"  # 文件已删除!
}
docker-remote-up() {
  docker-remote
  "$_ram_mount_docker_scripts/runall-remote-docker.sh" infra up    # 文件已删除!
}
alias dti='docker-infra-remote'    # 死别名
alias dru='docker-remote-up'       # 死别名
```

**无论选 A 还是 B，都应清理这些死代码。**

### 不属于 runAll：CI 脚本（2 个）

`ci/check_conf_sync.sh` 和 `ci/check_ddd_bdd_compliance.py` 是全域 CI 门禁——检查 task2app + valueStream + conf。放在 `scripts/ci/` 是正确的。

---

## 推荐操作

| # | 操作 | 影响 |
|---|------|------|
| 1 | `runall-local-promtail.sh` → `runAll/scripts/` | 文件移动 + 路径更新 |
| 2 | 清理 `docker-shell.sh` 死代码 | 移除 `dti`/`dru`/`docker-infra-remote`/`docker-remote-up` |
| 3 | docker-* 脚本：待用户决定 A 或 B | — |

## 变更汇总

| 仓库 | 新建 | 删除 | 修改 |
|------|------|------|------|
| scripts | — | `runall-local-promtail.sh` | `docker-shell.sh`（清理死代码） |
| runAll | `scripts/runall-local-promtail.sh` | — | — |
| conf | — | — | `runAll.yaml` 注释更新 |
