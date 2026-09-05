# Value Stream: scripts/ 目录清理

> Derived from design: `docs/design/scripts-cleanup-plan.md`
> Date: 2026-06-05

## Value Summary

开发者通过清理后的 scripts/ 目录获得更清晰的结构——删除死代码（docker-use-local, remote-compose-helper），将单服务 CI 脚本归位到所属服务（valueStream, taskGateway）。

## Related Value Streams

- **[2026-06-05-merge-conf-scripts-into-runall-value-stream](2026-06-05-merge-conf-scripts-into-runall-value-stream.md)**: 前置 — 已将 conf-* 脚本迁出 scripts/。本次是 scripts/ 目录的进一步精简。
- **关系类型**: extension — 在前置 conf 迁移的基础上，删除死代码并将剩余 CI 脚本归位。

## Value Increments

### Increment 1: 删除死代码 (Thin Slice)
**Value:** docker-use-local.sh 和 remote-compose-helper.sh 不再存在，消除误导。
**Scope:** 删除 2 文件 + 清理 docker-shell.sh 别名
**验证:** grep 全仓确认无引用

### Increment 2: CI 脚本归位
**Value:** check_go_ddd_compliance.py 归属 valueStream，taskGateway CI 脚本归属 taskGateway。
**Scope:** 移动 3 文件 + 更新 2 个引用方 + value-stream.yaml description
**验证:** CI 总入口 check_ddd_bdd_compliance.py 仍正常运行

## value-stream.yaml 变更
仅更新 description 文本（`scripts/ci/check_taskgateway_routes.sh` → `taskGateway/scripts/ci/check_routes.sh`）。
