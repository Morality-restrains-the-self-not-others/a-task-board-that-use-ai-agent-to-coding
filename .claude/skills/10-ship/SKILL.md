---
name: 10-ship
description: "Step 10 - Ship & Reflect: verify, merge to main, push origin, clean up, capture learnings."
---

# /10-ship — 交付与反思

验证通过后**直接合入 `main` 并推送 `origin`**，拆除 worktree、删除已落地 `feat/*`，并沉淀复盘。

## 默认交付路径（强制，2026-08-19 起）

**禁止**默认 `gh pr create` /「只开 PR、不直接 merge」。

| 场景 | 动作 |
|------|------|
| 在 `feat/*` 上开发 | `checkout main` → `merge`（或等价拣选）→ **`git push origin main`** |
| 已在 `main` 上开发（auto-flow 默认 SKIP worktree） | 提交后直接 **`git push origin main`** |
| 多仓 / submodule | **先**各子仓 push `origin/main`，**再** meta（规则 32） |
| 同主题仍有开放 PR | `gh pr close <n> --comment "已直接合入 main（ADR/约束：ship 默认不再开 PR）"` |

推送 `main` 成功后必须跑清理（见下节）。细则：`.ai/01_project_constraints/21_merged_feat_branch_cleanup.md`。

```bash
# 示例：单仓 feat → main → origin
git checkout main
git pull --ff-only origin main
git merge --no-ff feat/<name>   # 或已在 main 则跳过 merge
git push origin main

python3 runAll/scripts/cleanup_stale_worktrees.py --apply --shipped-branch feat/<name>
python3 runAll/scripts/delete_merged_feat_branches.py --apply
```

## 交付前验证清单（强制勾选）

在宣称已合入 / 公网已生效之前，按本次变更勾选：

- [ ] 相关测试 / 类型检查 / 约定验证已通过（或已记录跳过理由）
- [ ] **公网 SPA 构建** — 若改动了 `taskFE/app/src/` 中进入生产包的源码或 Vite 构建配置：已执行 `npm run build`（Vite build），且当前 spa 引用的 `/static/assets/main-*.js` 为 **200**（Django 已于 OPT-049 退役，collectstatic 不再需要）。不适用时写明 `n/a` 与理由。
- [ ] 已 **`git push origin main`**（含子仓优先顺序）
- [ ] 合入 `main` 后已拆除本轮 `{repo}-wt/` worktree 并删除已合入 / 等价落地的 `feat/*` 分支

依据：`task2app/front_project/ai.md`、`.claude/skills/simplify-and-harden/SKILL.md` Pass 4、`.ai/09_failure_experience/02_runtime_errors/04_public_spa_static_js_404_after_vite_build.md`。

## 合入 main 后：拆除 worktree 并删除 feat 分支（强制）

推送 `main` 成功后**必须**执行（禁止只建议「择机删除」）。**等价落地**（内容已在 main、SHA 不同）同样触发。先拆 worktree，再删分支：

```bash
python3 runAll/scripts/cleanup_stale_worktrees.py --apply --shipped-branch feat/<name>
python3 runAll/scripts/delete_merged_feat_branches.py --apply
```

## DDD Context (Backend Work)

Before shipping backend changes, invoke the `/6-ddd` skill in compliance audit mode
for the final DDD gate: confirm CI DDD/BDD compliance passes, all domain-layer tests pass without
infrastructure, all infrastructure tests pass, and domain events are properly wired
(publisher + consumer both present). **Additionally confirm every documented business intent
either publishes its mapped MQ event or has an explicit no-event exception in `docs/intents/`.**

## Archimate 架构状态迁移 — target → current（版本交付固化）

交付完成后，**必须**将已实现的目标架构切换为当前基线。核心原则：

> **老文件内容不动，只改状态标记。新文件清理临时标注后成为新的 current。**

### 1. 扫描待迁移的 target 架构

```bash
grep -rl "@status: target" docs/architecture/*.puml
```

- 没有 `@status: target` 的文件 → 本次交付不涉及架构变更，跳过后续步骤
- 存在 → 逐一完成以下迁移流程

### 2. 逐个视图迁移

对每个 `@status: target` 的文件执行：

**Step 2a — 将旧 current 标记为 archived（仅改一行元数据，内容不动）：**

```bash
# 找到所有 current 文件，逐个标记为 archived
for f in $(grep -rl "@status: current" docs/architecture/*.puml); do
  # 用 Edit 工具修改: '@status: current' → '@status: archived'
done
```

> ⚠️ **只改 `@status` 这一行**。文件其余内容（架构图、组件、关系）完全不动。这就是版本历史 — 随时可以打开查看当时的架构。

**Step 2b — 将 target 翻转为 current：**

用 Edit 工具修改新 target 文件：
```
旧: ' @status: target
新: ' @status: current

旧: ' @updated: <旧日期时间>
新: ' @updated: <今日日期时间 YYYY-MM-DD HH:MM>
```

**Step 2c — 清理临时变更标记：**

将 PlantUML 中的 `[NEW vN]`、`[MODIFIED vN]` 标记移除（变更已交付，不再是"新"）：

```
旧: note right of x "[NEW v3] 新增 CI/CD Runner Service" end note
新: note right of x "CI/CD Runner Service" end note

旧: note top of x "[MODIFIED v3] Kafka 替代 Redis" end note
新: note top of x "Kafka 替代 Redis 作为事件传输" end note
```

**Step 2d — 移除已废弃的元素：**

如果某个元素被标记为 `[DEPRECATED vN]`，从新 current 文件中**删除**该元素及其所有关联关系。同时移除对应的 note 标记。

**Step 2e — 移除变更图例：**

删除设计阶段添加的 `legend` 变更图例块（`Change Legend — v<N> 变更图例`），因为变更已交付。

**Step 2f — 验证修改后的架构文件：**

清理完成后，对修改过的 `.puml` 文件执行验证（同 brainstorming Step 3f）：
- PlantUML 语法检查（`@startuml`/`@enduml` 成对，alias 引用有效，rectangle 闭合）
- ArchiMate 模型验证（`mcp__archimate__validate_archimate_model`）
- 发现错误立即修正

更新伴生格式文件（`.diff.archimate` / `.full.archimate` / `.mermaid.md`）以反映删除废弃元素后的变更——两个 `.archimate`（增量 + 全量）都要同步并重新过 Archi `--loadModel` 验证。

### 3. 更新 VERSION_HISTORY.md

将 `docs/architecture/VERSION_HISTORY.md` 中本次版本条目更新为已交付：

```
旧:
## v3 🎯 target — OIDC SSO 认证桥接
- **状态**: 🎯 target（已设计，待交付）
- **交付日期**: —

新:
## v3 ✅ current — OIDC SSO 认证桥接
- **状态**: ✅ current（已交付）
- **交付日期**: 2026-08-19 15:00
```

### 4. 生成版本变更摘要

```markdown
## 🏛️ 架构版本交付 — v<N> → current

| 视图 | 旧 current (→ archived) | 新 current | 变更 |
|------|------------------------|-----------|------|
| Enterprise Landscape | `v<N-1>-xxx-<时间戳>-<作者>.puml` | `v<N>-xxx-<时间戳>-<作者>.puml` | +X 🟢 / ~Y 🟡 / -Z 🔴 |
| Application Integration | ... | ... | ... |

### 变更明细
- 🟢 新增: <列表>
- 🟡 修改: <列表>
- 🔴 移除: <列表>

### 历史文件（只读，可直接打开查看）
- `docs/architecture/v<N-1>-<视图>-<时间戳>-<作者>.puml` (archived)
- `docs/architecture/v<N-2>-<视图>-<时间戳>-<作者>.puml` (archived)
- ...

📋 版本历史已更新: `docs/architecture/VERSION_HISTORY.md`
```

### 5. 无 target 文件时

```
## 🏛️ 架构无变更

本次交付不涉及架构变更，架构基线保持不变。
```

## 完成后 — 下一步选择

交付完成后，使用 `AskUserQuestion` 工具让用户一键选择下一步：

```
header: "下一步"
question: "交付完成。接下来做什么？"
multiSelect: false
options:
  1. label: "开始新功能"
     description: "从头脑风暴开始下一个功能开发"
  2. label: "结束"
     description: "当前工作已完成，无需继续"
```

- 用户选 1 → 调用 `/1-brainstorming-design-docs`
- 用户选 2 → 流程结束，总结本次交付成果
