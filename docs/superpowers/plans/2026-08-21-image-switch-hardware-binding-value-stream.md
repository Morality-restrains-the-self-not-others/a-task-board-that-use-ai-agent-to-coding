# 切换镜像 × 硬件硬拦截 — 价值流

Mapping the approved design into a value stream.

## Related Value Streams

- `project-container-image-id` — 项目绑定已安装镜像；本增量在其上增加与 `server_run_template` 的同单一致性。
- `create-task-auto-run-backend-start` — 自动运行消费项目模版；硬拦截避免「新镜像 + 旧实例」进入 start-vm。

## Trigger → Value

用户在项目详情改默认容器镜像 → 系统拒绝不匹配组合或一次写入匹配的镜像+模版 → 后续自动运行/start-vm 使用与镜像 ISA 一致的实例。

## Increments（按价值排序）

1. **Thin slice（后端不变量）** — PATCH 先校验再写；跨架构 400；同架构/同单匹配 200。
2. **Core（项目 UI）** — 草稿镜像驱动硬件过滤；保存镜像带上完整模版；不兼容则无法保存。
3. **Essential（架构推导 + 规格启发式）** — `resolvePrimaryImageArchitecture` 以选中已安装镜像为准；`inferInstanceArchitecture` 不再误伤 `ecs.r6.*`。
4. **Enhancement** — 任务启动前端提示（后端 start-vm 已拦）。

## YAML

已写入 `conf/value-stream.yaml` step `project-image-hardware-arch-bind`。
