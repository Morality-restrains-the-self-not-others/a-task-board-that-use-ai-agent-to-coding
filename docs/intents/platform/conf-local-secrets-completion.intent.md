# Intent: conf-local 为唯一 overlay；加载器不读 config.local.yaml；历史泄露凭据轮换

## 背景与目标

ADR-0054 已把已跟踪 `conf/` YAML 的机密键抽到 gitignored `conf-local/`。加载器仍在 conf-local 之后合并 `config.local.yaml`，抽取留下的空键会覆盖真值。目标加载顺序只有：`conf/<app>/config.yaml` → `conf-local/<app>/config.yaml`。不保留对 `config.local.yaml` 的依赖。本机非机密覆盖也写入 conf-local。

## 范围与边界

- **范围内**：`shareLib/confload`、`runAll/scripts/conf_loader.py`、`gitService/scripts/load_gitservice_config.py`、抽取脚本、门禁、`conf-local.example`、`up-from-config-repo.sh`、文档、删除 `*Pwd.md`、轮换 runbook。
- **范围外**：不新造业务 API；不 filter-repo；无 OPS 窗口不改 live 控制台。

## 约束与风险

- 元规则 58 / ADR-0054；57；56。
- 三处加载器漏改一处仍会读 `.local.yaml`。
- 轮换 SSO JWT 须多服务同值。

## 验收标准

1. 磁盘上即使有 `config.local.yaml`，`ReadAppConfig` / `load_app_config` 结果不含其键（除非同键已在 conf 或 conf-local）。
2. 合并链只有 conf + conf-local。
3. 已跟踪无 `*Pwd.md`。
4. `conf-local.example` 覆盖已知相对路径（无真值）。
5. 轮换清单在 OPS 窗口完成。

## 实施计划

见 `docs/superpowers/specs/2026-08-31-conf-local-secrets-completion-design.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 只从 conf + conf-local 加载 | — | — | — | — | 配置加载，无领域事实 |
| 废弃已泄露凭据 | — | — | — | — | 控制面轮换 |

## 变更记录

- 2026-08-31：初稿；总体设计审批 approve；架构 v123
- 2026-08-31：修订加载顺序为仅两层，去掉 `config.local.yaml`
