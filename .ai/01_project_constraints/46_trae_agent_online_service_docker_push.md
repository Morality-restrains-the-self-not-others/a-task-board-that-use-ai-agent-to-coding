# 约束 46 — trae-agent 提交后必须推送 onlineServiceJS Docker 镜像（一级）

## 背景

任务容器运行的 online 镜像来自 `trae-agent/onlineServiceJS` 的 `buildDocker.sh` 推送到 registry。
公网/云任务容器**无** bind-mount 源码，只提交 `trae-agent` 而不推镜像会导致 registry 落后于源码，
新建或重启的任务容器仍跑旧二进制。

目录级 companion（`trae-agent/ai.md`、`trae-agent/onlineServiceJS/ai.md`）已要求提交后推送，
但此前**未提升为 monorepo 元规则**，亦无 Stop/SessionEnd 强制扫描与自动执行兜底，易被遗漏。

## 硬约束（一级，禁止忽略）

**当 `trae-agent` 发生会影响 online 镜像内容的变更并完成 git commit（或会话结束时仍有此类未推送变更）时，必须在目录：**

```text
<仓库根>/trae-agent/onlineServiceJS
```

**执行（命令原文，cwd 必须为上述目录）：**

```bash
DOCKER_PUSH=1 ./buildDocker.sh
```

- **默认**仅推送 `linux/amd64`。需 amd64+arm64 时：`DOCKER_PLATFORMS=all DOCKER_PUSH=1 ./buildDocker.sh`。
- **禁止**只提交代码却不推送镜像（仅文档 / 纯测试 / 明确不影响镜像内容的改动可跳过，须在交付说明写明原因）。
- **时机**：宜先 `git commit`（使 `arch_timestamp` 等 tag 与 HEAD 一致），再推送。
- **Agent**：在用户要求 commit / ship / 交付，或本会话完成 `trae-agent` 镜像相关提交后，**必须同步执行**上述命令（不得仅口头提醒）；失败须显式报告，禁止静默当作已交付。

### 影响镜像的路径（触发）

以下任一变更视为「影响镜像」（非穷尽，以 Dockerfile `COPY` / 构建上下文为准）：

- `trae-agent/onlineServiceJS/**`（业务源码、Dockerfile、依赖清单、docker 构建脚本等）
- `trae-agent/trae_agent/**`（打入镜像的 Python 包）
- `trae-agent` 根目录影响构建的文件（如 `pyproject.toml`、lockfile、与 Dockerfile 相关的上下文文件）

### 不触发

- 仅文档：`*.md` / `*.rst` / `docs/`
- 仅测试：`**/test/**`、`**/tests/**`、`**/__tests__/**`、`*.test.*`、`*_test.*`、`*.spec.*`
- 显式跳过：环境变量 `TRAE_AGENT_SKIP_DOCKER_PUSH=1`（须在交付说明写明原因）

## 强制机制（自动登记 + SessionEnd 兜底执行）

与约束 42（精准编译重启登记）对齐，由 Stop / SessionEnd hook 落地：

1. **扫描登记**：`scripts/lib/trae-agent-docker-push-scan.sh`（由 `auto-commit.sh` 每次触发调用）
   - 检测 `trae-agent` 工作树脏文件，或 HEAD 相对上次成功推送水位线（`.runall/trae_agent_docker_push_sha`）超前
   - 含镜像相关变更 → 写入 `.runall/trae_agent_docker_push_pending`（gitignore 运行时状态）
2. **执行推送**：`scripts/lib/trae-agent-docker-push.sh`（入口亦可 `scripts/trae-agent-docker-push.sh`）
   - Agent 提交后应**前台**调用（无 `--background`）
   - SessionEnd（非 checkpoint 阈值模式）若仍 pending：以 **`--if-pending --background`** 兜底启动推送，日志写入 `.runall/trae_agent_docker_push.log`（避免阻塞 hook 超时）；扫描/推送失败 **fail-open**，不阻断 auto-commit
3. **成功水位线**：推送成功后写入当前 `trae-agent` HEAD SHA 到 `.runall/trae_agent_docker_push_sha`，并清除 pending

## 验收标准

1. 修改 `trae-agent/onlineServiceJS` 源码后，Stop/SessionEnd 扫描使 `.runall/trae_agent_docker_push_pending` 出现。
2. Agent 在 commit 后于该目录执行 `DOCKER_PUSH=1 ./buildDocker.sh`，退出码 0；水位线 SHA 更新。
3. 仅改 `*.md` / 测试文件时不登记 pending。
4. `TRAE_AGENT_SKIP_DOCKER_PUSH=1` 时跳过执行并在 stderr 说明。
5. 自测：`bash scripts/lib/trae-agent-docker-push_selftest.sh` 全绿。

## 实现位置

| 组件 | 路径 |
|------|------|
| 约束专文 | `.ai/01_project_constraints/46_trae_agent_online_service_docker_push.md` |
| Cursor 元规则 | `.cursor/rules/trae-agent-docker-push.mdc` |
| 扫描 | `scripts/lib/trae-agent-docker-push-scan.sh` |
| 执行 | `scripts/lib/trae-agent-docker-push.sh`、`scripts/trae-agent-docker-push.sh` |
| Hook 集成 | `scripts/lib/auto-commit.sh` |
| 自测 | `scripts/lib/trae-agent-docker-push_selftest.sh` |
| 目录 companion | `trae-agent/ai.md`、`trae-agent/onlineServiceJS/ai.md` |

## 关联

- Companion：`trae-agent/ai.md`（目录级细则）
- 构建脚本：`trae-agent/onlineServiceJS/buildDocker.sh`
- 同类模式：约束 42 精准编译重启登记（登记 + 后续动作；本约束的后续动作是推镜像）
