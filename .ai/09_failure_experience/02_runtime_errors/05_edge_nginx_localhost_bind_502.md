# [运行时] 边缘 nginx 反代仅绑 127.0.0.1 的上游 → 502

## 现象

- 公网子域（`credential.api` / `aiendpoint` / `relay` / `businessapi` 等）返回 nginx `502 Bad Gateway`
- 同机 `curl http://127.0.0.1:PORT` 正常

## 根因

边缘 upstream 使用宿主机 **LAN IP:PORT**；进程若 `Listen(127.0.0.1:PORT)`，LAN 连不上。

## 修复原则

1. 需经边缘暴露的服务：`host: 0.0.0.0`（参考 `conf/ai/task-agent-support/config.yaml`）
2. 同机服务间 HTTP：固定 `http://127.0.0.1:PORT`（参考 Django `internalApiBase` / `taskCredentialServiceBase`）
3. 下发给浏览器/容器的 Origin 仍用 `${scheme}://${subdomains.*}`

> **升级为全服务硬约束（2026-07-15）**：不仅边缘暴露服务——monorepo **所有**服务禁止仅绑 `127.0.0.1`。见 `.ai/01_project_constraints/22_service_listen_host_not_localhost_only.md` 与根目录 `.ai.md`。同类 APISIX 案例：`13_apisix_host_docker_internal_localhost_bind.md`。

## 已覆盖（2026-07-12）

aiendpoint :8013、credential :8015、relay :8797、mock/businessapi :8796；TCG 同机 URL；runAll health 对 `0.0.0.0` 归一为 `127.0.0.1`。
