# 服务监听地址禁止仅绑 127.0.0.1

- **版本**：1.0.0
- **日期**：2026-07-15
- **适用范围**：monorepo 内**所有** HTTP/TCP 业务与基础设施服务（含 Go / Python / Node / sidecar；由 `conf/runAll.yaml` 编排或独立启动的进程均适用）

## 目标

防止智能体或人工以「更安全」为由将服务改为仅监听 loopback，导致：

- Docker 内 APISIX（upstream 常为 `host.docker.internal:PORT`）连不上宿主机进程；
- 边缘 nginx / 公网反代经 LAN IP 访问上游失败（502 / 超时 / 无法启动 OAuth 等）。

## 强制要求（禁止忽略）

1. **默认监听面**：服务进程的 listen / bind `host` **必须**为 **`0.0.0.0`**（或等价「所有网卡」写法，如 Go `":PORT"`、未指定 host 的 `http.ListenAndServe`）。
2. **禁止**：将运行时默认或 `conf/**/config.yaml` / provider YAML 中的服务 `host` 改为 **`127.0.0.1` / `localhost` 仅绑 loopback**。
3. **禁止**：以「安全加固」「最小暴露面」为由把已被 APISIX、边缘 nginx、Docker `host-gateway`、其它宿主机进程依赖的上游改回 loopback-only。
4. **配置与代码一致**：改默认值时须同步 `conf/`（含 provider 镜像副本）；禁止代码默认 `0.0.0.0` 而 conf 仍写 `127.0.0.1`（或相反）。
5. **Agent 义务**：审查、重构、迁移、hardening 时若发现或引入 `Listen(127.0.0.1…)` / `host: 127.0.0.1`，**必须改回 `0.0.0.0`**，不得合并、不得建议「生产再改」。

## 允许与禁止对照

| 场景 | 正确 | 错误 |
|------|------|------|
| 服务进程监听 | `0.0.0.0:PORT` / `:PORT` | `127.0.0.1:PORT` only |
| 同机服务间 **客户端** URL | `http://127.0.0.1:PORT/...` | 把被调服务改成只听 127.0.0.1 来「配合」 |
| APISIX Docker upstream | `host.docker.internal:PORT`（依赖宿主机 `0.0.0.0` 监听） | 上游仍指向 host-gateway，但进程只绑 loopback |
| 浏览器 / 公网 Origin | `${scheme}://${subdomains.*}` | 把 listen host 设成公网域名（listen ≠ advertise URL） |

## 验收自检（改 listen / 部署后必做）

```bash
# 1) 进程应对 *:PORT 或 0.0.0.0:PORT 监听（禁止仅 127.0.0.1:PORT）
ss -ltnp | rg ':<PORT>\\b'

# 2) 模拟 APISIX（Docker → 宿主机）
docker run --rm --add-host=host.docker.internal:host-gateway curlimages/curl:8.5.0 \
  -sf "http://host.docker.internal:<PORT><health_path>"
```

## 历史踩坑（交叉引用）

- 边缘 nginx → loopback 上游 502：`.ai/09_failure_experience/02_runtime_errors/05_edge_nginx_localhost_bind_502.md`
- APISIX `host.docker.internal` → taskGitOauth 仅绑 127.0.0.1（OAuth「无法启动授权」）：`.ai/09_failure_experience/02_runtime_errors/13_apisix_host_docker_internal_localhost_bind.md`

## 与相关规则的关系

- 不改变「同机调用走 `127.0.0.1`」的客户端约定（见失败经验 05）。
- 不改变防火墙 / 安全组 / APISIX 鉴权等边界控制；**暴露面由网络与网关收敛，不靠进程只绑 loopback。**
- 根目录元规则摘要：仓库根 [`.ai.md`](../../.ai.md)「服务监听地址」一节。
