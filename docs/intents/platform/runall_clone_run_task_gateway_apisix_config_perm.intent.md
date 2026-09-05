# Intent: clone-run 重启 task-gateway 时 APISIX 须能改写 bind-mount 的 config.yaml

## 背景与目标

用 `.daydaymoney-deploy-seed` / `daydaymoney-deploy` 在新目录部署后，http://192.168.1.10:9999/ 「全部重启」对 `task-gateway` 失败：

`READINESS_TIMEOUT` — `Get "http://127.0.0.1:18081/api/health/": dial tcp 127.0.0.1:18081: connect: connection refused`

Docker 已 `Created`/`Started`，但容器随即 `Exited (1)`。apache/apisix standalone 入口在 uid 636 下执行：

`echo "$(sed … config.yaml)" > config.yaml`

源码仓 `/tmp/ram-work` 上该文件常为 `0666`（可写）；seed/rsync 后为 git 默认 `0644`、属主 1000，重定向 EACCES → 进程退出 → 宿主机 18081 拒绝连接，run.sh 仍空等 180s。

目标：`run.sh start` 在 `compose up` 前把 `apisix/config.yaml` 变成 uid 636 可写；容器已退出时不得再空等满超时。

## 范围与边界

- **范围内**：`taskGateway/run.sh` 启动前准备 bind-mount；契约测例；clone-run 部署树与源码仓同一 `run.sh`。
- **范围外**：APISIX 上游 502、路由内容、与 ram-work 抢同一 compose 项目名（同机双根部署属运维拓扑，另跟踪）。

## 约束与风险

- 不得把私钥 `certs/*.pem` 改成 world-readable。
- 不得把 `0666` 提交进 git；只在启动时 chmod。
- 源码仓与 seed 的 `run.sh` 必须同一逻辑，禁止只改部署树。

## 验收标准

1. `prepare_apisix_logs_for_start`（或其所在启动准备函数）对 `apisix/config.yaml` 执行 `chmod a+rw`（host 与/或 docker `--user 0`）。
2. 临时树里 `config.yaml` 为 `0644` 时，调用该准备步骤后 mode 含 other-write。
3. `compose up` 后若 `taskgateway-apisix-1` 未在跑，start 路径打印容器日志且 ready 等待不再按默认 180s 空转。
4. 隔离端口下 seed 配置可启动 APISIX 并对 health URL 建立 TCP（不再 connection refused）。

## 实施计划

1. 扩展 `prepare_apisix_logs_for_start`：chmod `apisix/config.yaml`。
2. start：compose up 后探测容器是否仍 running，否则缩短等待并 dump logs。
3. 契约 + 功能测例（0644 → writable）。
4. 在 clone-run 树验证 `curl -sf http://127.0.0.1:18081/api/health/`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| clone-run 重启时 APISIX 能改写 config.yaml | — | — | — | — | 运维网关容器启动路径，不产生业务领域事件 |

## 变更记录

- 2026-09-01：初稿。seed 部署全部重启 task-gateway READINESS_TIMEOUT / connection refused。
