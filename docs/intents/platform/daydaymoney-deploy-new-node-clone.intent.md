# Intent: 新节点克隆 daydaymoney-deploy 不只靠 conf-local

## 背景与目标

v123 将机密 overlay 收口为 `$DEPLOY_ROOT/conf-local/`（与 `conf/` 同相对路径）。运维口号写成「新机器只拷 conf-local」。实际 clone-run 还依赖 GitHub Release 产物、网关/OIDC PEM、`INFRA_HOST`、`GITHUB_TOKEN`，且 `.daydaymoney-deploy-seed` 的 `up.sh` 仍按已撤销的 `*.local.yaml` / HOST_SECRETS 树加载。

目标：机密面（YAML + 网关 TLS + OIDC 签名钥 + 已有支付 PEM）全部在 `conf-local/`；产物面手拷 `deploy-binaries/` → `artifacts/`。seed 与源码仓 `up.sh` 对齐。

## 范围与边界

- **范围内**：seed re-export、README 清单、网关/OIDC PEM 迁 `conf-local/`、`run.sh`/`jwt.go` 改读路径、架构 v124、新 Release 钉、文档/意图。
- **范围外**：不改 YAML 两步合并规则；不把 datadir 纳入 clone。

## 约束与风险

- ADR-0052 / ADR-0054；元规则 42 / 58。
- 现网 Release `deploy-20260831` 早于 confload 两步合并。

## 验收标准

1. 网关 TLS 与 OIDC 签名 PEM 的 SSOT 在 `conf-local/`；`DEPLOY_MODE` 缺失则失败（不现签/不 mint）。
2. re-seed 后配置仓 `up.sh` 只 overlay `conf-local/`；gitignore 含 `/conf-local/`；无 HOST_SECRETS.md。
3. README：clone → 拷 conf-local + `deploy-binaries/`→`artifacts/` → `up.sh`。

## 实施计划

见 `docs/superpowers/specs/2026-08-31-daydaymoney-deploy-new-node-clone-design.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 新节点 clone-run 装配 | — | — | — | — | 运维交付，不产生业务领域事实 |

## 变更记录

- 2026-08-31：初稿（头脑风暴）
- 2026-08-31：修订 — PEM 迁入 conf-local
- 2026-08-31：`up.sh` 拉 Release 强制 HTTP/1.1（GitHub HTTP/2 PROTOCOL_ERROR）
- 2026-08-31：GitHub 仍下不了产物 → 源码仓归集 `deploy-binaries/`，新节点手拷到 `artifacts/`，`up.sh` 默认跳过 GitHub
- 2026-08-31：提交钩子增量归集 `deploy-binaries/`（`SKIP_COLLECT_DEPLOY_BINARIES=1` 可关）
- 2026-08-31：`taskEvents-bin.tar.gz` 缺 `project_deleted` worker 导致 clone-run 启动 60s 超时；collect 改为 live `taskEvents/bin` 优先重打，install 对照 `INTENT_PATHS` 缺 worker 即失败
- 2026-08-31：install 覆盖正在运行的 ELF 改为旁路 `mv`，避免 `cp` 报 ETXTBSY /「文本文件忙」
