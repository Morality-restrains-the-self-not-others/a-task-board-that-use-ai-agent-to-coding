# 实施计划: scripts/ 目录清理

> 输入: `docs/design/scripts-cleanup-plan.md`
> DDD: skipped (no new business concepts)
> NFR: skipped (cleanup, no runtime change)

---

## 任务清单

### Item 1: 删除 docker-use-local.sh + 清理别名

- [ ] **1.1** 删除 `scripts/docker-use-local.sh`
- [ ] **1.2** 从 `scripts/docker-shell.sh` 移除:
  - `docker-local()` 函数 (~line 14-16)
  - `docker-local-up()` 函数 (~line 36-39)
  - `alias dl='docker-local'` (~line 44)
  - `alias dlu='docker-local-up'` (~line 48)
  - `case` 中 `local` 分支 (~line 56-58)
- [ ] **1.3** 验证: `grep -rn 'docker-use-local\|docker-local-up\|alias dl=' scripts/docker-shell.sh` 无结果

### Item 2: 删除 remote-compose-helper.sh

- [ ] **2.1** 删除 `scripts/remote-compose-helper.sh`
- [ ] **2.2** 验证: `grep -rn 'remote-compose-helper' --include='*' | grep -v '.git/' | grep -v 'docs/'` 无结果

### Item 3: check_go_ddd_compliance.py → valueStream/

- [ ] **3.1** 移动 `scripts/ci/check_go_ddd_compliance.py` → `valueStream/scripts/ci/check_go_ddd_compliance.py`
- [ ] **3.2** 移动 `scripts/ci/README_GO_DDD_COMPLIANCE.md` → `valueStream/scripts/ci/README.md`
- [ ] **3.3** 更新 `scripts/ci/check_ddd_bdd_compliance.py`:
  - 第 53 行: `"scripts" / "ci"` → `"valueStream" / "scripts" / "ci"`
  - 第 79 行: 错误信息路径同上
- [ ] **3.4** 验证: `python3 scripts/ci/check_ddd_bdd_compliance.py` Go DDD 检查仍运行

### Item 4: taskGateway CI 脚本 → taskGateway/

- [ ] **4.1** 移动 `scripts/ci/check_taskgateway_routes.sh` → `taskGateway/scripts/ci/check_routes.sh`
- [ ] **4.2** 移动 `scripts/ci/smoke_taskgateway.sh` → `taskGateway/scripts/ci/smoke.sh`
- [ ] **4.3** 更新 `taskGateway/scripts/ci/smoke.sh` 内部的 check_routes 路径
- [ ] **4.4** 更新 `value-stream.yaml:1669` description 文本
- [ ] **4.5** 验证: `bash taskGateway/scripts/ci/check_routes.sh` 可执行

---

## 文件变更汇总

| 仓库 | + | - | ~ |
|------|---|---|---|
| scripts | 0 | 6 | 2 |
| valueStream | 2 | 0 | 0 |
| taskGateway | 2 | 0 | 0 |
| root | 0 | 0 | 1 (value-stream.yaml) |
