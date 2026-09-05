# 功能意图：runAll 独立小组管理各区域 GitLab

> 状态: **implemented** | 日期: 2026-08-18

## 背景与目标

http://10.2.150.68:9999/ 的 `infrastructure` 组把现网 `git-service` 与 MySQL/Redis/Kafka 混在一起，且没有上海 `git-service-tencent-sh-1` 条目。运维无法在同一小组内启停/探活不同区域的 GitLab。

## 范围与边界

- 仅改 `conf/runAll.yaml` 分组与编排条目，以及 Prometheus file_sd 同步。
- 不改 GitLab 容器内部配置。现网走 `gitService/run.sh`；上海走 `runall_ssh_sh_gitlab.sh` → `ssh sh` + `/opt/daydaymoney/gitservice-tencent-sh-1` 的 `docker compose`。
- 不把 GitLab 迁机；SH-1 实例在 Host sh（探活 `1.117.67.121:8014`）。禁止在 INFRA 机本地执行 `deploy_tencent_sh_1.sh`。

## 约束与风险

- 探活端口必须等于 conf SSOT：现网 8012、SH-1 8014，禁止互相拷贝。
- GitLab 可插拔：除 `git-service*` 自身外，其它服务不得 `depends_on: git-service*`（ADR-0048）。分组搬迁仍不得改名。
- `gitlab-regions` 必须 `skip_start_all: true`：start-all 不启动 GitLab；9999 面板单启 / 按组启动仍可用。
- 两实例必须设置不同 `GITSERVICE_CONF_APP` / `COMPOSE_PROJECT_NAME`，避免打到同一容器。

## 意图 → 事件

| 意图 | 事件 | 发布点 | 消费者 | MQ类型/契约 |
|------|------|--------|--------|-------------|
| 在 9999 按组启停区域 GitLab | 无对应事件 | — | — | 纯编排/运维，无业务状态变更 |

## 验收标准

1. 9999 出现独立组 `gitlab-regions`
2. 组内至少含 `git-service` 与 `git-service-tencent-sh-1`
3. `infrastructure` 不再包含 `git-service`
4. 两服务启动命令与健康检查端口不同（8012 vs 8014）
5. start-all 计划不含 `git-service*`；`PlanStartGroup("gitlab-regions")` 仍含两实例

## 实施计划

1. 新增 `gitlab-regions` 组并迁入/补齐条目
2. 再生 `runall-health-targets.json`
3. `TestProductionConfig_GitlabRegionsGroup` 锁定
