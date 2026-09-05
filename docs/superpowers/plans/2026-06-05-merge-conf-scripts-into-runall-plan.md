# 实施计划: conf-read/sync 脚本合并进 runAll

> 输入:
> - 设计文档: `docs/design/merge-conf-scripts-into-runall.md`
> - 价值流: `docs/superpowers/plans/2026-06-05-merge-conf-scripts-into-runall-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-05-merge-conf-scripts-into-runall-nfr-clarification.md`
> - DDD: skipped (无新业务概念)

---

## 任务清单

### 阶段 1: conf_loader 提取 + 脚本迁移

- [ ] **T1.1** 创建 `runAll/scripts/conf_loader.py`
  - 从 `task2app/Saas_project/config/conf_loader.py` 提取: `JSON_KEY_TO_APP_DIR`, `monorepo_root`, `load_app_config`, `build_runtime_snapshot`, `load_domain_events_config`, `load_git_oauth_catalog`
  - 复用 `conf_lib.py` 的 `load_yaml`, `deep_merge`（消除重复，自包含，无 task2app 依赖）
  - 文件: `runAll/scripts/conf_loader.py`

- [ ] **T1.2** 迁移 5 个脚本文件到 `runAll/scripts/`
  - `scripts/conf-read.py` → `runAll/scripts/conf-read.py`
  - `scripts/conf-sync.py` → `runAll/scripts/conf-sync.py`
  - `scripts/conf_lib.py` → `runAll/scripts/conf_lib.py`
  - `scripts/conf-sync-all.sh` → `runAll/scripts/conf-sync-all.sh`
  - `scripts/migrate_port_config_to_conf.py` → `runAll/scripts/migrate_port_config_to_conf.py`

- [ ] **T1.3** 更新 `runAll/scripts/conf-read.py` 的 import
  - 移除: `sys.path.insert(0, str(ROOT / "task2app" / "Saas_project"))`
  - 移除: `from config.conf_loader import ...`
  - 新增: `sys.path.insert(0, str(Path(__file__).resolve().parent))` + `from conf_loader import ...`
  - 文件: `runAll/scripts/conf-read.py`

- [ ] **T1.4** 更新 `runAll/scripts/conf-sync-all.sh` 内部路径
  - `python3 "$ROOT/scripts/conf-sync.py"` → `python3 "$ROOT/runAll/scripts/conf-sync.py"`
  - 文件: `runAll/scripts/conf-sync-all.sh`

- [ ] **T1.5** 更新 `runAll/src/conf_sync.go` 的脚本路径
  - 第 31 行: `filepath.Join(root, "scripts", "conf-sync-all.sh")` → `filepath.Join(root, "runAll", "scripts", "conf-sync-all.sh")`
  - 第 39 行: `bash scripts/conf-sync-all.sh` → `bash runAll/scripts/conf-sync-all.sh`
  - 文件: `runAll/src/conf_sync.go`

- [ ] **T1.6** 验证 thin slice
  - 运行: `python3 runAll/scripts/conf-read.py snapshot-json` 对比迁移前输出
  - 运行: `bash runAll/scripts/conf-sync-all.sh` 确认 exit 0
  - 运行: `cd runAll && go test ./... -run TestSyncMonorepoConf` 确认通过

### 阶段 2: 外部调用方路径更新

- [ ] **T2.1** 更新 8 个 sync.sh 的脚本路径
  - `conf/core/django/sync.sh`: `$ROOT/scripts/conf-sync.py` → `$ROOT/runAll/scripts/conf-sync.py`
  - `conf/frontend/vue/sync.sh`: 同上
  - `conf/auth/task-auth/sync.sh`: 同上
  - `conf/auth/git-oauth/sync.sh`: 同上
  - `conf/events/domain-events/sync.sh`: 同上
  - `conf/ai/ai-provider/sync.sh`: 同上
  - `conf/gateway/task-sse/sync.sh`: 同上

- [ ] **T2.2** 更新 `taskSSE/src/config.mjs` 的脚本路径
  - 第 67 行: `scripts/conf-read.py` → `runAll/scripts/conf-read.py`
  - 第 78 行: `scripts/conf-read.py` → `runAll/scripts/conf-read.py`
  - 文件: `taskSSE/src/config.mjs`

- [ ] **T2.3** 更新 `scripts/ci/check_conf_sync.sh` 的脚本路径
  - 第 33 行: `./scripts/conf-sync-all.sh` → `./runAll/scripts/conf-sync-all.sh`
  - 文件: `scripts/ci/check_conf_sync.sh`

- [ ] **T2.4** 更新 `conf/README.md` 的文档路径
  - `scripts/conf-read.py` → `runAll/scripts/conf-read.py`
  - `scripts/conf-sync-all.sh` → `runAll/scripts/conf-sync-all.sh`
  - 文件: `conf/README.md`

- [ ] **T2.5** 验证阶段 2
  - 逐一执行 8 个 sync.sh，确认均 exit 0
  - 运行: `bash scripts/ci/check_conf_sync.sh` 确认通过
  - 运行: `cd runAll && go test ./...` 全量测试通过

### 阶段 3: 旧文件清理 + 全仓验证

- [ ] **T3.1** 删除旧脚本文件
  - `rm scripts/conf-read.py`
  - `rm scripts/conf-sync.py`
  - `rm scripts/conf_lib.py`
  - `rm scripts/conf-sync-all.sh`
  - `rm scripts/migrate_port_config_to_conf.py`
  - `rm -rf scripts/__pycache__/`

- [ ] **T3.2** 全仓扫描确认无残留引用
  - 运行: `grep -rn 'scripts/conf-read\|scripts/conf-sync\|scripts/conf_lib' --include='*.py' --include='*.sh' --include='*.mjs' --include='*.go' --include='*.yaml' | grep -v 'docs/' | grep -v '.git/' | grep -v '__pycache__'`
  - 预期输出: 空（排除 docs/ 历史文档）

- [ ] **T3.3** 最终综合验证
  - 运行: `cd valueStream && go test ./...` 确认无 regression
  - 运行: `python3 scripts/ci/check_ddd_bdd_compliance.py` 确认 CI 门禁通过
  - 确认 taskSSE 可正常启动: `node taskSSE/src/index.mjs`（需运行环境）

---

## 文件变更汇总

| 操作 | 数量 | 文件 |
|------|------|------|
| 新建 | 1 | `runAll/scripts/conf_loader.py` |
| 迁移 | 5 | conf-read.py, conf-sync.py, conf_lib.py, conf-sync-all.sh, migrate_port_config_to_conf.py |
| 修改 | 13 | conf_sync.go, 8×sync.sh, config.mjs (2处), check_conf_sync.sh, conf/README.md |
| 删除 | 5+1 | 5 个旧脚本 + `__pycache__/` |

## 依赖图

```
T1.1 (conf_loader.py)
  └→ T1.2 (迁移 5 文件)
       └→ T1.3 (更新 conf-read import)
            └→ T1.4 (更新 conf-sync-all.sh)
                 └→ T1.5 (更新 conf_sync.go)
                      └→ T1.6 (验证 thin slice)
                           └→ T2.1-T2.4 (并行: 8 sync.sh + taskSSE + CI + README)
                                └→ T2.5 (验证阶段 2)
                                     └→ T3.1 (删除旧文件)
                                          └→ T3.2 (全仓扫描)
                                               └→ T3.3 (最终验证)
```
