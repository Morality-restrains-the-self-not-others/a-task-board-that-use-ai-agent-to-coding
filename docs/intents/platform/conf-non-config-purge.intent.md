# Intent: conf.git 只跟踪配置及允许的薄工具

## 背景与目标

`/tmp/ram-work/conf` 是独立子仓，却被全仓钩子分发、表征测试并置、以及 ramsync plist / 区域 compose / generateKey 等历史文件填进非配置资产。目标：已跟踪文件符合允许清单（YAML + companion + sync.sh + 仓身份 + 一支配置质量 pre-commit），其余迁出或删除。

## 范围与边界

- **范围内**：`conf.git` 跟踪面、`deploy_repo_random_precommit.sh` 对 conf 的 SKIP、CI 允许清单与子仓钩子豁免、迁出 compose/测试/脚本/plist、瘦身 `.githooks`。
- **范围外**：不改业务 API；不搬 `value-stream.yaml` / `runAll.yaml`；不新开 ADR；不写架构 target；不 filter-repo。

## 约束与风险

- 元规则 42 / 47 / 29；ADR-0052 配方不在 YAML 树。
- conf 子仓提交看不到 meta hooks → oauth live check 必须留在 conf 薄 pre-commit。
- 子仓优先提交（规则 32）：先 push conf，再 meta 指针。

## 验收标准

1. 允许清单门禁对当前树绿。
2. conf 无 plist、无 docker-compose、无 `test_*.py`、无模板钩子全家桶。
3. 分发器再次执行不会把 `.githooks/lib` 或 `.claude/settings.json` 写回 conf。
4. git-oauth YAML 暂存时 conf pre-commit 仍调用 live check。

## 实施计划

见 `docs/superpowers/specs/2026-09-01-conf-non-config-purge-design.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 收紧 conf.git 跟踪面 | — | — | — | — | 仓库布局，无领域事实 |

## 变更记录

- 2026-09-01：初稿（/1-brainstorming）
- 2026-09-01：总体设计审批 approve；不写架构 target
