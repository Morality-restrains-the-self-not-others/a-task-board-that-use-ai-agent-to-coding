# [运行时] relay overlay 白名单漏挂新 mjs 导致二进制预览缺模块

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-20
- 最后修改：2026-07-20
- 维护者：Trae AI 团队

## 现象

本地 selected_image 已 overlay `server.mjs`（含 `import './layerFileContent.mjs'`），但容器内无该文件 → 进程启动失败或 `/files/*` 无法返回 `kind=binary` 精确属性。

## 根因

`go_relayToTrae` 曾按文件白名单挂载 `server.mjs` / `layerFs.mjs` / `layerGitOauthPush.mjs`，**新模块不会自动挂载**。公网路径则完全依赖镜像 `COPY onlineServiceJS`，未推镜像的旧容器同样缺文件。

## 解决方案

1. 本地：整目录挂载 `onlineServiceJS/src` → `/app/onlineServiceJS/src:ro`（见 `appendOnlineServiceSrcOverlayMounts`）。
2. 公网：commit 后 `DOCKER_PUSH=1 ./buildDocker.sh`，并重启/重建任务容器。

## 预防

- 禁止再对 `src/` 使用易漏网的文件白名单 overlay。
- 改 onlineServiceJS 交付时对照 companion `onlineServiceJS/ai.md`「源码进容器路径」。

## 关联

- `go_relayToTrae/src/container_image_overlay.go`
- `trae-agent/onlineServiceJS/Dockerfile`（`COPY onlineServiceJS`）
- `OPT-20260720-016`
