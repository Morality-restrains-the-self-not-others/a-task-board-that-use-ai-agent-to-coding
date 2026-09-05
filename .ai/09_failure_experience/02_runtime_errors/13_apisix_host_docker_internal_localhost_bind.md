# [运行时] APISIX host.docker.internal 上游仅绑 127.0.0.1 → OAuth/网关失败

## 基本信息

- 案例编号：FE-20260715-APISIX-LOOPBACK
- 录入日期：2026-07-15
- 最后更新：2026-07-15
- 关联服务：taskGitOauth（:8002）、taskGateway APISIX `up-gitOauth`

## 失败现象

- 项目详情页提示「授权异常」「无法启动 OAuth 授权」
- 浏览器经网关调用 `/api/accounts/*/oauth/start-from-gateway/` 失败（无 `authorize_url`）
- 同机 `curl http://127.0.0.1:8002/api/health/` 正常；Docker 内访问 `host.docker.internal:8002` 失败

## 根因

1. APISIX（Docker）upstream 为 `host.docker.internal:8002`（见 `taskGateway/apisix/apisix.yaml` `up-gitOauth`）。
2. 「安全加固」将 taskGitOauth / provider YAML 的 `host` 改为仅 **`127.0.0.1`**。
3. host-gateway 流量到达宿主机非 loopback 接口 → 连接拒绝；网关无法转发 OAuth 启动。

## 修复

1. 监听与 conf 恢复 **`host: 0.0.0.0`**（代码默认与 `conf/auth/git-oauth/providers/*.yaml` 等一致）。
2. 重启服务；用 Docker curl 验收 `host.docker.internal:PORT`。
3. **制度**：全服务禁止再绑 127.0.0.1-only，见 `.ai/01_project_constraints/22_service_listen_host_not_localhost_only.md` 与根目录 `.ai.md`。

## 预防

- Agent hardening / 迁移时不得把 listen 改成 loopback-only。
- 变更 listen 后必跑：`ss -ltnp` + Docker `host.docker.internal` 探活。
