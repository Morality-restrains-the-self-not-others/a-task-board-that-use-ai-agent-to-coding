# 价值流：二进制部署与独立配置仓

- **Date:** 2026-08-30
- **Design:** `docs/superpowers/specs/2026-08-30-binary-deploy-config-repo-design.md`
- **Domain:** 平台交付 / 运维

## Related Value Streams

- `runall-precise-restart` / `runall-cascade-lifecycle` / 重启-编译分离：本流 **扩展** — 部署模式把 compile-then-swap 换成 download-then-swap；开发机精准编译不变。
- `runall-orchestrator-independence`：正交（编排器退出策略）。

无冲突：不改租户业务字段。

## 价值主张

运维可以在**没有源码树**的主机上，用配置仓 + 钉版本产物拉起平台；改端口不必碰源码仓。

## Increments（按价值排序）

| # | 增量 | 用户可感知价值 | 本会话交付 |
|---|------|----------------|------------|
| I1 | `FindConfigRoot`：`CONF_ROOT`/`DEPLOY_ROOT` 优先 | 进程能在无 `.gitmodules` 的根上读 conf | **是（P0）** |
| I2 | `releases.yaml` 钉解析 + deploy-sync 不编译 | 错版本不 exec；失败保留 last-good | **是（P0 契约）** |
| I3 | 配置仓骨架 + 双写说明 | 运维知道往哪 clone | 文档 + example；真仓创建若 `gh` 可用则建私有空仓 |
| I4 | 源码仓 `conf/` → `conf.example/` | 源码不再带生产拓扑 | **否（P3，现网双写未切）** |
| I5 | 独立部署机无源码验收 | 成功标准 1 | **否（P4，需新主机）** |

## Fields impact

无 `<service>.<table>.<column>` 业务字段。配置平面：`releases.yaml` artifacts 键、环境变量 `CONF_ROOT`/`DEPLOY_ROOT`。

## Test files

- `shareLib/confload/load_test.go` — FindConfigRoot
- `runAll/src/deploy_sync_test.go`（或 `scripts` 单测）— 钉解析 / 禁止 go build

## Status

I1–I2 `active`（本会话）。I3 骨架 `active`。I4–I5 `planned`。
