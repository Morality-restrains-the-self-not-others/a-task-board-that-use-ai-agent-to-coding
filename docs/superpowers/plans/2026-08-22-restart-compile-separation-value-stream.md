# 价值流：重启与编译分离

## Related Value Streams

- `runall-cascade-lifecycle` / `runall-global-start-stop-all` — 启停链；本增量规定 **start/restart 不编译**
- `runall-build-all-progress` / `runall-build-group-progress` — 唯一全量编译入口（不变）
- `runall-precise-restart-reload-yaml` — 精准重启前热加载 YAML；本增量把重启语义改为 compile-then-swap

非 greenfield：修改既有运维流。

## Increments

1. **restart-does-not-compile** — RestartAll / Start / 单服务 ↻ 不执行 build_command
2. **task-events-start-no-build** — `run.sh start` 只 exec last-good
3. **precise-restart-compile-then-swap** — 先编后切；失败保留旧进程

## YAML

已追加到 `conf/value-stream.yaml` 的 `runall-cascade-lifecycle` 与 `runall-precise-restart-reload-yaml`。
