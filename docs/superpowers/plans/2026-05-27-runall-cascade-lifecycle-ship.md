# Ship & Reflect: runAll 链式启停

## 交付验证

- [x] `go test ./...` in `runAll/` — PASS
- [x] 设计 / 价值流 / NFR / DDD / 实施计划 / 审查文档齐全
- [x] `value-stream.yaml` 已追加 `runall-cascade-lifecycle` 流（`planned` 步骤）

## 手工验收清单（`http://localhost:9999/`）

1. 仅点击 `taskFE`「启动」→ `git-oauth` → `saas-backend` → `taskFE` 依次 healthy
2. 下游运行时点击 `git-oauth`「关闭」→ 先停 `taskFE` / `ai-provider`，再停上游
3. `platform` 组「启动本组」按依赖顺序拉起
4. API `cascade: false` 关闭上游时仍报 active downstream（`ui_test` 已覆盖）

## PR / 合并

**阻塞：** 当前目录 `/Users/task2app/gitClone/ramDisk/ram-mount` 不是 git 仓库，无法执行 `gh pr create`。

建议用户本地：

```bash
cd /path/to/ram-mount   # 你的 git 远程仓库根
git checkout -b feat/runall-cascade-lifecycle
git add runAll/ value-stream.yaml docs/superpowers/
git commit -m "$(cat <<'EOF'
feat(runall): cascade start/stop and start-group on Web UI

One-click service lifecycle follows depends_on: start pulls upstream,
stop tears down downstream, plus start-group for platform-style stacks.
EOF
)"
git push -u origin HEAD
gh pr create --title "runAll: 链式启停与启动本组" --body "见 docs/superpowers/plans/2026-05-27-runall-cascade-lifecycle-value-stream.md"
```

## 经验摘要

- 复用 `runnerRuntimeContextRepository` + 纯函数计划生成，避免 duplicate `stopOrderForGroup` 逻辑漂移。
- 首步失败不包装 `CascadeFailure`，保持与既有 API 错误兼容。
- 全局启停留二期，MVP 已覆盖最高频「单行一键」场景。
