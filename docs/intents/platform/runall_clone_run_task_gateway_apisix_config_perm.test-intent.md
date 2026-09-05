# 测试意图: clone-run 重启 task-gateway 时 APISIX 须能改写 bind-mount 的 config.yaml

## 测试目标

证明 `run.sh` 在 compose up 前把 standalone 入口要重写的 `apisix/config.yaml` 变成非属主（uid 636）可写；容器已退出时不空等 180s。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 脚本契约 | `prepare_apisix_logs_for_start` 含对 `apisix/config.yaml` 的 `chmod a+rw` |
| 功能 | 临时目录 `0644` 文件经 `prepare-host-mounts` 后 other-write 位置位 |
| 脚本契约 | start 分支在 compose up 后检查容器是否 running，未运行则缩短 ready 等待 |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| GW-1 | `run.sh` 准备函数 | 读源 | 函数体含 `apisix/config.yaml` 与 `chmod a+rw` |
| GW-2 | 临时 `taskGateway/apisix/config.yaml` mode `0644` | `bash run.sh prepare-host-mounts` | 退出 0，mode 含 `0o002` |
| GW-3 | start 分支 | 读源 | compose up 之后有容器名探测；未 running 时 ready 超时被收成短等待 |

## 数据与环境

- 测例用 `tmp_path`，不改 live `/tmp/ram-work/taskGateway/apisix/config.yaml` 的 git 模式。
- `prepare-host-mounts` 可能调用 docker chmod；docker 不可用时 host `chmod a+rw` 仍须使测例通过。

## 通过标准

```bash
python3 taskGateway/scripts/test_run_sh_apisix_ready_wait.py
```

## 业务意图 → 事件对照（测试）

运维网关启动路径无 MQ 事件；不断言 Kafka。
