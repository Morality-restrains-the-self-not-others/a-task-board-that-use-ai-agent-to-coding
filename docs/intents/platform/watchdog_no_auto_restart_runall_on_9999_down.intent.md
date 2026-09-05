# Intent: :9999 不可用时看门狗不自动重启 runAll

## 背景与目标

`runAll/scripts/ensure_services_healthy.py` 由 crontab 每 5 分钟运行。OPT-20260813-006 曾在 `:9999` 失联时自动 `setsid` 拉起 `bin/runAll` 并 `POST /api/start-all`。热替换 / 人工停 UI 窗口内，这会再拉起一个编排器并全量 start-all，与正在进行的 shutdown-self 或全部重启打架。

目标：看门狗在 `:9999` 不可用时**只告警、不自动重启 runAll**。需要旧行为时显式 `--restart-runall`，且 `.runall/no_restart_runall` 存在时仍禁止拉起。

## 范围与边界

- **范围内**：watchdog 对 runAll UI（`:9999`）的恢复策略；crontab 条目带 `--no-restart-runall`；运行时禁止文件 `.runall/no_restart_runall`。
- **范围外**：对 `conf/runAll.yaml` 中 platform 服务端口失活时的自愈（task-auth 等）不变。

## 约束与风险

- 禁止文件与 `--no-restart-runall` 优先于 `--restart-runall`。
- 默认关闭自动重启后，runAll UI 宕机需人工 `runAll/run.sh`（或显式 opt-in）。
- crontab 已有条目必须改写，不能因「已存在」而留下旧的无 flag 行。

## 验收标准

1. `:9999` 失联且未传 `--restart-runall` 时不调用 `spawn_runall`。
2. `.runall/no_restart_runall` 存在时，即使 `restart=True` 也不拉起。
3. `install_cron` 写入的 crontab 行含 `--no-restart-runall`。
4. 对 platform 服务的端口失活自愈逻辑不被这次改动关闭。

## 实施计划

1. `recover_runall` 读取禁止文件；`restart` 默认 false
2. argparse：`--no-restart-runall` 为默认 cron 显式开关；`--restart-runall` 为 opt-in
3. `install_cron` 更新已有 watchdog 行
4. 单测覆盖默认不拉起 / 禁止文件 / crontab 行

## 业务意图 → 事件对照

> 运维 crontab watchdog，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: :9999 不可用时看门狗不自动重启 runAll | — | — | — | — | 运维本地 cron/文件门禁，不产生业务领域事件 |

## 变更记录

- 2026-08-17：默认与 crontab 禁止 `:9999` 失联时自动重启 runAll + start-all；可用 `.runall/no_restart_runall` 运行时通知。
