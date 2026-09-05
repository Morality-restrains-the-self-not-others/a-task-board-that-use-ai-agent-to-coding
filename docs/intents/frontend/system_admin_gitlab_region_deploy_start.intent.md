# 前端：系统管理 GitLab 区域卡片启动命令按实例区分

> 状态: **implemented** | 日期: 2026-08-18

## 背景与目标

`/system-admin/gitlab-resources` 区域卡片「启动命令」曾对所有实例硬编码 `bash gitService/run.sh start`。现网 `git-service` 与上海 `git-service-tencent-sh-1` 是不同服务，复制同一条命令会打到错误实例。

## 范围与边界

- 展示字段 `service_start` 由 `taskBill` 按已解析的 `service_process` / conf 目录生成。
- 前端只渲染 API 值，不在 Vue 侧推导启动命令。
- 不启停容器、不改 `gitService/run.sh`。

## 约束与风险

- 现网启动命令须与 `conf/runAll.yaml` `git-service.start_command` 一致。
- SH-1 须指向 `gitService/scripts/deploy_tencent_sh_1.sh`（含 `GITSERVICE_CONF_APP` / OIDC / compose project）。

## 意图 → 事件

| 意图 | 事件 | 发布点 | 消费者 | MQ类型/契约 |
|------|------|--------|--------|-------------|
| 管理员查看区域部署路径 | 无对应事件 | — | — | 纯展示/只读，无服务端状态变更 |

## 验收标准

1. `tencent-shanghai-5`（legacy `git-service`）启动命令为 `bash gitService/run.sh start`
2. `tencent-sh-1` 启动命令为 `bash gitService/scripts/deploy_tencent_sh_1.sh`
3. 其它 slug 为 `GITSERVICE_CONF_APP=git-service-<slug> bash gitService/run.sh start`
4. 两张现网卡片的启动命令文本不相同

## 实施计划

1. `gitlabRegionServiceStart` 按服务名生成命令
2. Go / Vitest 回归锁定差异
3. 精准编译重启 `task-bill`（API 字段变化；FE 已透传 `service_start`）
