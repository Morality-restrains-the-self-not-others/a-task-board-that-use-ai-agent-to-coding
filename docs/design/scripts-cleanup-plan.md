# scripts/ 目录清理设计

> 头脑风暴产物 — 设计文档
> 日期: 2026-06-05
> 状态: 待审批

---

## 清理候选总览

| # | 目标 | 问题 | 处置 |
|---|------|------|------|
| 1 | `docker-use-local.sh` | 已废弃（exit 1），`dl`/`dlu` 别名指向死路径 | **删除** + 清理 docker-shell.sh |
| 2 | `remote-compose-helper.sh` | 零调用方，全仓搜索无引用 | **直接删除** |
| 3 | `ci/check_go_ddd_compliance.py` | 仅检查 valueStream，可归于该服务 | **移至** `valueStream/scripts/ci/` |
| 4 | `ci/{check_taskgateway_routes,smoke_taskgateway}.sh` | 仅检查 taskGateway，可归于该服务 | **移至** `taskGateway/scripts/ci/` |

---

## 逐项分析

### 1. docker-use-local.sh — 删除

**现状**：
- 脚本内容：打印警告 "不再推荐本地 Docker Desktop"，`exit 1`
- 被 `docker-shell.sh` 的 `docker-local()` 函数和 `dl`/`dlu` 别名指向
- `docker_env_ensure_local_context()`（在 docker-env.sh）是真正的工作函数，被 `runall-local-promtail.sh` 调用

**调用链分析**：
```
docker-shell.sh → docker-local() → docker-use-local.sh → exit 1 (永远失败)
                → dl 别名 → 同上
                → dlu 别名 → docker-local-up → docker-local → 同上

runall-local-promtail.sh → docker_env_ensure_local_context() → 正常工作 ✓
```

**结论**：`docker-use-local.sh` 和 `dl`/`dlu` 别名都已失效。`runall-local-promtail.sh` 不经过它。

**操作**：
1. 删除 `scripts/docker-use-local.sh`
2. 从 `docker-shell.sh` 移除:
   - `docker-local()` 函数
   - `docker-local-up()` 函数
   - `alias dl='docker-local'`
   - `alias dlu='docker-local-up'`
   - `case "${RAM_MOUNT_DOCKER_DEFAULT:-remote}"` 中的 `local` 分支
   - 保留 `docker-remote()` / `dr` / `dru` / `ds` / `dt` / `dti`

### 2. remote-compose-helper.sh — 删除

**现状**：
- 46 行，是 `docker-desktop-helper.sh`（94 行）的 Linux 极简替代
- `grep -rn 'remote-compose-helper'` 全仓搜索：**零结果**（排除 docs/）
- 设计意图是 sync 到远程 Linux 执行，但从未被任何脚本或配置引用

**结论**：死代码，可直接删除。

**操作**：
1. 删除 `scripts/remote-compose-helper.sh`

### 3. ci/check_go_ddd_compliance.py — 移至 valueStream/

**现状**：
- 检查 `valueStream/domain/**/*.go` 的 DDD 合规性
- 被 `scripts/ci/check_ddd_bdd_compliance.py`（CI 总入口）调用
- 有独立 README: `scripts/ci/README_GO_DDD_COMPLIANCE.md`

**调用链**：
```
scripts/ci/check_ddd_bdd_compliance.py
  ├── go_target = root / "scripts" / "ci" / "check_go_ddd_compliance.py"   # 第 53 行
  └── "expected scripts/ci/check_go_ddd_compliance.py\n"                    # 第 79 行
```

**操作**：
1. 移动 `scripts/ci/check_go_ddd_compliance.py` → `valueStream/scripts/ci/check_go_ddd_compliance.py`
2. 移动 `scripts/ci/README_GO_DDD_COMPLIANCE.md` → `valueStream/scripts/ci/README.md`
3. 更新 `scripts/ci/check_ddd_bdd_compliance.py` 第 53 行: `"scripts" / "ci"` → `"valueStream" / "scripts" / "ci"`
4. 更新第 79 行错误信息同样
5. 更新 `.claude/skills/` 中 3 个 SKILL.md 的路径引用

### 4. ci/{check_taskgateway_routes,smoke_taskgateway}.sh — 移至 taskGateway/

**现状**：
- `check_taskgateway_routes.sh`: 被 `smoke_taskgateway.sh` 调用，被 `value-stream.yaml` 的 description 文本引用
- `smoke_taskgateway.sh`: 无外部调用方

**调用链**：
```
scripts/ci/smoke_taskgateway.sh:6 → bash "$ROOT/scripts/ci/check_taskgateway_routes.sh"
value-stream.yaml:1669 → description: "scripts/ci/check_taskgateway_routes.sh --check 通过"
```

**操作**：
1. 移动 `scripts/ci/check_taskgateway_routes.sh` → `taskGateway/scripts/ci/check_routes.sh`
2. 移动 `scripts/ci/smoke_taskgateway.sh` → `taskGateway/scripts/ci/smoke.sh`
3. 更新 `smoke.sh` 内部的 `check_taskgateway_routes.sh` 路径引用
4. 更新 `value-stream.yaml:1669` 的 description 文本

---

## 变更汇总

| 仓库 | 新建 | 删除 | 修改 |
|------|------|------|------|
| scripts | — | `docker-use-local.sh`, `remote-compose-helper.sh`, `ci/check_go_ddd_compliance.py`, `ci/README_GO_DDD_COMPLIANCE.md`, `ci/check_taskgateway_routes.sh`, `ci/smoke_taskgateway.sh` | `docker-shell.sh`, `ci/check_ddd_bdd_compliance.py` |
| valueStream | `scripts/ci/check_go_ddd_compliance.py`, `scripts/ci/README.md` | — | — |
| taskGateway | `scripts/ci/check_routes.sh`, `scripts/ci/smoke.sh` | — | — |
| root | — | — | `value-stream.yaml` (description 文本) |
| `.claude/skills/` | — | — | 3 个 SKILL.md 路径更新 |

---

## 风险

| 风险 | 等级 | 缓解 |
|------|------|------|
| `dl`/`dlu` 别名被开发者脚本依赖 | 低 | 别名已 exit 1，依赖者早已无法使用 |
| `remote-compose-helper` 被外部 cron/CI 引用 | 低 | 全仓扫描零结果；脚本名称不在任何配置中 |
| taskGateway CI 路径变更后 CI 断裂 | 低 | `smoke_taskgateway.sh` 无外部调用方；`value-stream.yaml` 仅 description 文本 |

---

## 价值流影响

无新增价值流。`value-stream.yaml:1669` 的 description 文本需更新路径。
