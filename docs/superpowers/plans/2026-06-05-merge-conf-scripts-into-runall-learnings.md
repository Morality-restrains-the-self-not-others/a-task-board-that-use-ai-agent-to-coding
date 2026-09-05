# 经验记录: conf-read/sync 脚本合并进 runAll

> 日期: 2026-06-05
> 流水线: 1→3→4→5(D)→6→7→8→9

## 交付成果

| 仓库 | 变更 | 文件数 |
|------|------|--------|
| runAll | conf_loader.py 新建, 5 脚本迁移, conf_sync.go 路径更新 | +6, ~1 |
| scripts | 5 脚本删除, check_conf_sync.sh 路径更新 | -5, ~1 |
| conf | 7 sync.sh + README.md 路径更新 | ~8 |
| taskSSE | config.mjs 路径更新 | ~1 |

## 关键教训

### 1. repo_root() 的路径深度陷阱
**问题**: 从 `scripts/` 迁移到 `runAll/scripts/` 后，`conf_lib.py` 的 `repo_root()` 仍用 `parent.parent`，导致定位到 `runAll/` 而非 monorepo root。
**修复**: 改为自适应向上遍历，查找 `conf/core/django/config.yaml` 标记文件。
**教训**: 依赖 `__file__` 相对路径的函数在文件迁移时须重新验证。

### 2. conf-sync-all.sh 的 ROOT 计算同理
**问题**: shell 脚本的 `dirname $0/..` 在新的 `runAll/scripts/` 位置只能到 `runAll/`。
**修复**: `dirname $0/../..` 三阶向上。
**教训**: Bash 和 Python 的路径推导逻辑都须在迁移后逐一验证。

### 3. conf_loader 路径需与 conf/ 目录重组对齐
**问题**: 新 conf_loader 的路径（如 `auth/git-oauth`、`events/domain-events`）需要与 conf/ 二级目录重组后的结构匹配。
**验证**: 通过 `conf-read.py core/django --json` 的输出确认路径正确。

### 4. 预存在的测试失败不应阻止交付
- `conf-sync-all.sh`: `docker-infra/config.yaml` 缺失（非本次变更引起）
- `valueStream/src`: 已弃用的 value stream 状态（非本次变更引起）
**判断**: 我们的变更引入 0 个新失败。

## 文件变更总结

```
新建 1:  runAll/scripts/conf_loader.py
迁移 5:  runAll/scripts/{conf-read,conf-sync,conf_lib,conf-sync-all.sh,migrate_port_config_to_conf}.py
修改 8:  sync.sh ×7 + conf/README.md (conf/)
修改 1:  scripts/ci/check_conf_sync.sh (scripts/)
修改 1:  taskSSE/src/config.mjs (taskSSE/)
修改 2:  runAll/src/conf_sync.go + runAll/scripts/conf_lib.py (runAll/)
删除 6:  scripts/{conf-read,conf-sync,conf_lib,conf-sync-all.sh,migrate_port_config_to_conf}.py + __pycache__/
```
