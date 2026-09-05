# ADR-0022: 公网 taskFE 由 Docker nginx 静态常驻，Vite 只负责构建

- **Status:** accepted
- **Date:** 2026-08-20
- **Author:** cursor
- **Deciders:** 工程团队（Docker nginx 方案已批准并实施）

---

## Context

公网 SPA 经 APISIX `spa-catch-all` 反代到应用主机 `:4000`。runAll 当前用 Node `vite preview`。该进程被 stop-all / pkill 后，APISIX 约 160ms 内 **502**。原子 `dist` 替换不能在「没有监听者」时提供 HTML。

应用主机 **没有** 系统 `nginx` 命令。用户选定 **Docker nginx + volume**，与已有 APISIX 容器同机，避免 apt 装系统 nginx。

约束：监听 `0.0.0.0`（或 compose 发布到 `0.0.0.0:4000`）；可调配置落 `conf/frontend/vue/`；hashed 资源缺失须真 404。

Docker 若 bind-mount **已解析的 symlink 目标**，容器在 start 时钉死旧目录，之后宿主机切 `html` **不生效**。必须 mount **含 symlink 的父目录**。

## Decision

We will serve production/runAll taskFE with a **Docker nginx container** (`restart: unless-stopped`):

- Host publishes **`0.0.0.0:4000` → container `:80`**. APISIX `host.docker.internal:4000` stays.
- Bind-mount `taskFE/app/public` (parent of `releases/` + `html` symlink). nginx `root` follows `html`.
- Vite is **build-only**. A successful build atomically retargets `html` and **does not** recreate the container.
- Compose `image` / `mem_limit` / `ports` consume `conf/frontend/vue/config.yaml` only.
- Precise restart of taskFE is build + symlink (reload only if nginx.conf changed).
- `npm run dev` remains Vite HMR and is mutually exclusive with port 4000.

## Alternatives Considered

### Alternative 1: Keep vite preview

- **Pros:** 无新容器
- **Cons:** stop 即 502
- **Why rejected:** 不满足静态常驻

### Alternative 2: 宿主机 `nginx -p`（上一版提案）

- **Pros:** 与其它 runAll 进程同为宿主机；reload 简单
- **Cons:** 需 apt；与系统 nginx 配置易搅在一起
- **Why rejected:** 用户明确改选 Docker nginx 挂卷

### Alternative 3: bind-mount 仅 `html` 指向的当前 release

- **Pros:** 挂载面小
- **Cons:** 切 symlink 后容器仍服务旧 release
- **Why rejected:** 发版静默失败

### Alternative 4: SH 边缘 nginx 直接 root dist

- **Why rejected:** dist 不在边缘机

## Consequences

### Positive

- 不依赖宿主机 nginx 包；`unless-stopped` 比 Node preview 更能扛 runAll 抖动
- 发版只切 symlink，容器 Id 不变 → 公网 SPA 不 502
- APISIX 上游地址零改动

### Negative / Trade-offs

- 多一个 compose 服务与镜像拉取
- 本机 HMR 与容器抢 4000
- 须正确 mount 父目录，否则「构建成功但公网仍旧包」

### Mitigations

- 启动脚本断言挂载点内 `html` 在容器中仍是 symlink
- companion 写明：HMR 前先 `docker compose stop`
- `releaseKeep` 默认 2

## References

- 设计: [2026-08-20-taskfe-nginx-static-resident-design.md](../superpowers/specs/2026-08-20-taskfe-nginx-static-resident-design.md)
- 意图: [taskfe_nginx_static_resident.intent.md](../intents/backend/taskfe_nginx_static_resident.intent.md)
